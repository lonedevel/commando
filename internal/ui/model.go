// Package ui implements commando's interactive option form.
package ui

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"

	"github.com/lonedevel/commando/internal/cmdline"
	"github.com/lonedevel/commando/internal/manpage"
	"github.com/lonedevel/commando/internal/shellword"
	"github.com/lonedevel/commando/internal/store"
)

// Config is how the program was invoked.
type Config struct {
	Line       string // initial command line, e.g. "ls -la ~/src"; may be empty
	UseCache   bool
	PreferLong bool
	Output     *termenv.Output // for OSC52 clipboard fallback
	Store      *store.Store    // presets and history; nil disables them
	Confirm    bool            // ask before running a command that uses risky options
}

// Result is what the user decided.
type Result struct {
	Command  string
	Accepted bool
	Key      string // the command the form was for, e.g. "git commit"
}

type mode int

const (
	modePick mode = iota
	modeLoading
	modeForm
	modeManual
	modeSub // choosing a subcommand of a tool such as git
)

type rowKind int

const (
	rowArgs rowKind = iota
	rowHeader
	rowGroup
	rowOpt
)

type row struct {
	kind  rowKind
	opt   int // option index for rowOpt
	group int // group index for rowGroup / radio rows, else -1
	text  string
}

type loadedMsg struct {
	spec    *manpage.Spec
	rest    []cmdline.Word
	err     error
	elapsed time.Duration
	cmd     string
	subs    []manpage.Subcommand // the tool's commands, when it was opened without one
}

type statusClearMsg int

type pathCmdsMsg []string

// Model is the Bubble Tea model.
type Model struct {
	cfg  Config
	mode mode
	w, h int

	spin spinner.Model

	// pick mode
	pick       textinput.Model
	pathCmds   []string
	sugg       []string
	suggSel    int
	suggRecent bool // sugg holds recent command lines, not command names
	pickErr    string
	loadWhat   string

	// form mode
	spec          *manpage.Spec
	values        []cmdline.Value
	inputs        map[int]*textinput.Model // per text-like option
	args          textinput.Model
	argInputs     []*textinput.Model // one per positional argument, when the usage line was readable
	warnedMissing bool               // Enter was pressed once with required arguments empty
	filter        textinput.Model
	filtering     bool
	rows          []row
	vis           []int // indexes into rows
	haystack      []string
	cursor        int // index into vis
	offset        int // first visible list line
	dropdown      bool
	ddSel         int
	customEdit    int // option index whose dropdown value is being typed, or -1
	helpScroll    int
	preferLong    bool
	listTop       int   // screen row of first list line (for mouse)
	lineRows      []int // vis index per rendered list line (for mouse)

	// presets & recent (shown in place of the option list)
	lib        bool
	libItems   []libItem
	libSel     int
	naming     bool     // typing a name for a new preset
	confirming []string // risky options awaiting "run anyway?" confirmation
	presetIn   textinput.Model

	// subcommand browser
	toolMsg   loadedMsg // the tool itself, as loaded, to reopen its own form
	subs      []manpage.Subcommand
	subItems  []subItem
	subSel    int
	subOff    int
	subFilter textinput.Model
	fromSubs  bool   // the form was opened from the browser; Esc goes back to it
	subPicked string // the subcommand line being loaded from the browser

	// manual mode
	manBack   mode // where Esc returns to
	manLines  []string
	manOff    int
	manSearch textinput.Model
	manFind   bool
	manQuery  string

	status    string
	statusErr bool
	statusID  int
	loadInfo  string

	result Result
}

// New creates the model.
func New(cfg Config) *Model {
	m := &Model{cfg: cfg, preferLong: cfg.PreferLong, customEdit: -1}
	m.spin = spinner.New(spinner.WithSpinner(spinner.MiniDot))
	m.spin.Style = sCursor

	m.pick = newInput("e.g. ls, grep, tar, curl, git commit")
	m.pick.Prompt = "❯ "
	m.pick.PromptStyle = sCursor
	m.args = newInput("files, URLs, other arguments…")
	m.filter = newInput("type to filter options")
	m.filter.Prompt = ""
	m.presetIn = newInput("e.g. long listing with sizes")
	m.presetIn.Prompt = ""
	m.subFilter = newInput("type a command name or what it does")
	m.manSearch = newInput("search manual")
	m.manSearch.Prompt = "/"

	if strings.TrimSpace(cfg.Line) == "" {
		m.mode = modePick
		m.pick.Focus()
	} else {
		m.mode = modeLoading
		m.loadWhat = strings.Fields(cfg.Line)[0]
	}
	return m
}

func newInput(placeholder string) textinput.Model {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = placeholder
	ti.TextStyle = sValue
	ti.PlaceholderStyle = sFaint
	ti.Cursor.Style = sCursor
	km := textinput.DefaultKeyMap
	km.CharacterForward.SetKeys("right")
	km.CharacterBackward.SetKeys("left")
	km.AcceptSuggestion.SetKeys()
	km.NextSuggestion.SetKeys()
	km.PrevSuggestion.SetKeys()
	km.DeleteCharacterForward.SetKeys("delete")
	ti.KeyMap = km
	return ti
}

// Result returns the outcome once the program has exited.
func (m *Model) Result() Result { return m.result }

// Init implements tea.Model.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{m.spin.Tick, textinput.Blink}
	if m.mode == modeLoading {
		cmds = append(cmds, m.load(m.cfg.Line))
	} else {
		cmds = append(cmds, scanPath)
	}
	return tea.Batch(cmds...)
}

// load resolves the command (and subcommand) and parses its manual.
func (m *Model) load(line string) tea.Cmd {
	useCache := m.cfg.UseCache
	return func() tea.Msg {
		start := time.Now()
		words := cmdline.Split(line)
		if len(words) == 0 {
			return loadedMsg{err: fmt.Errorf("no command given")}
		}
		vals := make([]string, len(words))
		for i, w := range words {
			vals[i] = w.Value
		}
		spec, n, err := manpage.LoadLine(context.Background(), vals, useCache)
		rest := words[min(n, len(words)):]
		name := strings.Join(vals[:max(1, n)], " ")
		var subs []manpage.Subcommand
		if err == nil && n == 1 && len(rest) == 0 {
			subs = manpage.Subcommands(spec, useCache)
		}
		return loadedMsg{spec: spec, rest: rest, err: err, elapsed: time.Since(start), cmd: name, subs: subs}
	}
}

func scanPath() tea.Msg {
	seen := map[string]bool{}
	var out []string
	for _, dir := range strings.Split(os.Getenv("PATH"), string(os.PathListSeparator)) {
		ents, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, e := range ents {
			n := e.Name()
			if seen[n] || strings.HasPrefix(n, ".") {
				continue
			}
			seen[n] = true
			out = append(out, n)
		}
	}
	return pathCmdsMsg(out)
}

func (m *Model) setStatus(s string, isErr bool) tea.Cmd {
	m.status, m.statusErr = s, isErr
	m.statusID++
	id := m.statusID
	return tea.Tick(3*time.Second, func(time.Time) tea.Msg { return statusClearMsg(id) })
}

// Update implements tea.Model.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.resizeInputs()
		return m, nil
	case spinner.TickMsg:
		if m.mode != modeLoading {
			return m, nil
		}
		var cmd tea.Cmd
		m.spin, cmd = m.spin.Update(msg)
		return m, cmd
	case statusClearMsg:
		if int(msg) == m.statusID {
			m.status = ""
		}
		return m, nil
	case pathCmdsMsg:
		m.pathCmds = msg
		m.updateSuggestions()
		return m, nil
	case loadedMsg:
		return m, m.onLoaded(msg)
	case tea.MouseMsg:
		return m, m.onMouse(msg)
	case tea.KeyMsg:
		if msg.String() == "ctrl+c" {
			m.result = Result{}
			return m, tea.Quit
		}
		switch m.mode {
		case modePick:
			return m, m.updatePick(msg)
		case modeForm:
			return m, m.updateForm(msg)
		case modeManual:
			return m, m.updateManual(msg)
		case modeSub:
			return m, m.updateSubs(msg)
		}
	}
	// Forward cursor blink and other messages to the focused input.
	return m, m.forwardToFocused(msg)
}

func (m *Model) onLoaded(msg loadedMsg) tea.Cmd {
	if msg.err != nil && m.fromSubs {
		cmd := m.openSubs()
		return tea.Batch(cmd, m.setStatus(msg.err.Error(), true))
	}
	if len(msg.subs) > 0 {
		// A tool opened without a command: choose one first.
		m.toolMsg = msg
		m.toolMsg.subs = nil
		m.subs = msg.subs
		m.subFilter.SetValue("")
		return m.openSubs()
	}
	if msg.err != nil {
		m.mode = modePick
		m.pickErr = msg.err.Error()
		m.pick.SetValue(m.cfg.Line)
		if m.cfg.Line == "" {
			m.pick.SetValue(msg.cmd)
		}
		m.pick.CursorEnd()
		return tea.Batch(m.pick.Focus(), scanPath)
	}
	m.spec = msg.spec
	m.fill(msg.rest)
	m.buildRows()
	m.mode = modeForm
	m.cursor = 0
	m.toFirstOption() // the argument fields stay just above
	m.resizeInputs()
	src := "man page"
	if m.spec.Source == "help" {
		src = "--help"
	}
	m.loadInfo = fmt.Sprintf("%d options from %s in %s", len(m.spec.Options), src, msg.elapsed.Round(time.Millisecond))
	m.manLines = colorizeManual(m.spec.Manual)
	var note tea.Cmd
	if m.subPicked != "" && msg.cmd != m.subPicked {
		// No option list for the command itself: its name stays as an argument.
		note = m.setStatus("No option list for "+m.subPicked+", so this is "+msg.cmd+"'s form with the command as an argument", false)
	}
	m.subPicked = ""
	// Offer saved presets and recent commands when starting from scratch.
	if len(msg.rest) == 0 && m.loadLibrary() > 0 {
		m.lib = true
		m.libSel = 1 // the first preset (or latest command); Blank form is one ↑ away
	}
	return tea.Batch(m.focusCurrent(), note)
}

// fill sets the form from command-line words that follow the command.
func (m *Model) fill(rest []cmdline.Word) {
	vals, args := cmdline.Prefill(m.spec, rest)
	m.values = vals
	m.inputs = map[int]*textinput.Model{}
	for i, o := range m.spec.Options {
		if o.Kind == manpage.KindFlag {
			continue
		}
		ti := newInput(strings.ToLower(o.Arg))
		if m.values[i].Text != "" {
			ti.SetValue(m.values[i].Text)
		}
		m.inputs[i] = &ti
	}
	// Radio groups: keep only the last selected member.
	for _, g := range m.spec.Groups {
		last := -1
		for _, mi := range g.Members {
			if m.values[mi].On {
				last = mi
			}
		}
		for _, mi := range g.Members {
			m.values[mi].On = mi == last
		}
	}
	m.argInputs = nil
	if len(m.spec.Args) > 0 {
		parts := distributeArgs(m.spec.Args, shellword.Split(args))
		for i, a := range m.spec.Args {
			ph := strings.ToLower(a.Name)
			if a.Repeat {
				ph += " …"
			}
			ti := newInput(ph)
			ti.SetValue(parts[i])
			ti.CursorEnd()
			m.argInputs = append(m.argInputs, &ti)
		}
		args = ""
	}
	m.args.SetValue(args)
	m.args.CursorEnd()
	m.resizeInputs()
}

// distributeArgs assigns typed words to argument fields in order: single
// arguments take one word each, a repeatable one takes what the arguments
// after it don't need, and anything left over goes to the last field.
func distributeArgs(args []manpage.Arg, words []shellword.Word) []string {
	out := make([]string, len(args))
	k := 0
	for i, a := range args {
		n := 1
		if a.Repeat {
			after := 0
			for _, b := range args[i+1:] {
				if !b.Repeat {
					after++
				}
			}
			n = max(0, len(words)-k-after)
		}
		var raw []string
		for ; n > 0 && k < len(words); n-- {
			raw = append(raw, words[k].Raw)
			k++
		}
		out[i] = strings.Join(raw, " ")
	}
	for ; k < len(words); k++ {
		out[len(out)-1] = strings.TrimSpace(out[len(out)-1] + " " + words[k].Raw)
	}
	return out
}

// argsText is the positional-argument text, in order.
func (m *Model) argsText() string {
	if len(m.argInputs) == 0 {
		return m.args.Value()
	}
	var parts []string
	for _, ti := range m.argInputs {
		if v := strings.TrimSpace(ti.Value()); v != "" {
			parts = append(parts, v)
		}
	}
	return strings.Join(parts, " ")
}

// argInput returns the text field for an Arguments row.
func (m *Model) argInput(r *row) *textinput.Model {
	if r.opt >= 0 && r.opt < len(m.argInputs) {
		return m.argInputs[r.opt]
	}
	return &m.args
}

// argSpec returns the positional argument an Arguments row edits, if any.
func (m *Model) argSpec(r *row) (manpage.Arg, bool) {
	if r != nil && r.kind == rowArgs && r.opt >= 0 && r.opt < len(m.spec.Args) {
		return m.spec.Args[r.opt], true
	}
	return manpage.Arg{}, false
}

// missingArgs lists required arguments that are still empty.
func (m *Model) missingArgs() []string {
	var names []string
	for i, a := range m.spec.Args {
		if a.Required && i < len(m.argInputs) && strings.TrimSpace(m.argInputs[i].Value()) == "" {
			names = append(names, a.Name)
		}
	}
	return names
}

func (m *Model) blurInputs() {
	m.args.Blur()
	for _, ti := range m.argInputs {
		ti.Blur()
	}
	for _, ti := range m.inputs {
		ti.Blur()
	}
}

// buildRows lays out the form: sections, radio groups and options.
func (m *Model) buildRows() {
	s := m.spec
	m.rows = []row{{kind: rowArgs, opt: -1, group: -1}}
	m.haystack = []string{""}
	if len(s.Args) > 0 {
		m.rows, m.haystack = nil, nil
		for i := range s.Args {
			m.rows = append(m.rows, row{kind: rowArgs, opt: i, group: -1})
			m.haystack = append(m.haystack, "")
		}
	}
	placed := map[int]bool{}
	cur := "\x00"
	nsections := map[string]bool{}
	for _, o := range s.Options {
		nsections[o.Section] = true
	}
	for i, o := range s.Options {
		if placed[i] {
			continue
		}
		if o.Section != cur {
			cur = o.Section
			title := cur
			if len(nsections) == 1 || title == "" || title == "Description" {
				title = "Options"
			}
			m.rows = append(m.rows, row{kind: rowHeader, text: title, group: -1})
			m.haystack = append(m.haystack, "")
		}
		if g := s.GroupOf(i); g >= 0 {
			grp := s.Groups[g]
			m.rows = append(m.rows, row{kind: rowGroup, group: g, text: grp.Label})
			m.haystack = append(m.haystack, "")
			for _, mi := range grp.Members {
				placed[mi] = true
				m.rows = append(m.rows, row{kind: rowOpt, opt: mi, group: g})
				m.haystack = append(m.haystack, m.hay(mi)+" "+strings.ToLower(grp.Label))
			}
			continue
		}
		m.rows = append(m.rows, row{kind: rowOpt, opt: i, group: -1})
		m.haystack = append(m.haystack, m.hay(i))
	}
	m.applyFilter()
}

func (m *Model) hay(i int) string {
	o := &m.spec.Options[i]
	return strings.ToLower(strings.Join(o.Names, " ") + " " + o.Arg + " " + o.Label + " " + firstPara(o.Desc) + " " + o.Section)
}

func firstPara(s string) string {
	p, _, _ := strings.Cut(s, "\n\n")
	return p
}

func (m *Model) applyFilter() {
	q := strings.Fields(strings.ToLower(m.filter.Value()))
	var cur int = -1
	if m.cursor < len(m.vis) {
		cur = m.vis[m.cursor]
	}
	m.vis = m.vis[:0]
	match := make([]bool, len(m.rows))
	for i, r := range m.rows {
		if r.kind != rowOpt {
			continue
		}
		ok := true
		for _, t := range q {
			if !strings.Contains(m.haystack[i], t) {
				ok = false
				break
			}
		}
		match[i] = ok
	}
	for i, r := range m.rows {
		switch r.kind {
		case rowArgs:
			m.vis = append(m.vis, i)
		case rowOpt:
			if match[i] {
				m.vis = append(m.vis, i)
			}
		case rowHeader, rowGroup:
			// show if any option before the next header (or group end) matches
			for j := i + 1; j < len(m.rows); j++ {
				if m.rows[j].kind == rowHeader || (r.kind == rowGroup && m.rows[j].group != r.group) {
					break
				}
				if match[j] {
					m.vis = append(m.vis, i)
					break
				}
			}
		}
	}
	// keep the cursor on the same row if still visible
	m.cursor = 0
	for k, ri := range m.vis {
		if ri == cur {
			m.cursor = k
		}
	}
	if len(q) > 0 {
		// Jump to the best match: exact names beat prefixes beat labels.
		best, bestScore := -1, -1
		for k, ri := range m.vis {
			if m.rows[ri].kind != rowOpt {
				continue
			}
			if sc := m.score(m.rows[ri].opt, strings.Fields(m.filter.Value())); sc > bestScore {
				best, bestScore = k, sc
			}
		}
		if best >= 0 {
			m.cursor = best
		}
	}
	m.helpScroll = 0
}

// score ranks an option against the filter words: exact names first (a
// same-case match beating one that differs only in case, so -X finds -X,
// not -x), then name prefixes, then names and labels containing the word.
func (m *Model) score(i int, raw []string) int {
	o := &m.spec.Options[i]
	label := strings.ToLower(o.Label)
	total := 0
	for _, r := range raw {
		t := strings.ToLower(r)
		best := 1
		for _, orig := range o.Names {
			n := strings.ToLower(orig)
			switch {
			case orig == r || strings.TrimLeft(orig, "-") == r:
				best = max(best, 120)
			case n == t || strings.TrimLeft(n, "-") == t:
				best = max(best, 100)
			case strings.HasPrefix(strings.TrimLeft(n, "-"), strings.TrimLeft(t, "-")):
				best = max(best, 50)
			case strings.Contains(n, t):
				best = max(best, 20)
			}
		}
		if strings.Contains(label, t) {
			best = max(best, 10)
		}
		total += best
	}
	return total
}

func (m *Model) curRow() *row {
	if m.cursor < 0 || m.cursor >= len(m.vis) {
		return nil
	}
	return &m.rows[m.vis[m.cursor]]
}

func focusable(r row) bool { return r.kind == rowArgs || r.kind == rowOpt }

// textInputFor returns the text input that should receive typing for the
// current row, or nil.
func (m *Model) textInputFor(r *row) *textinput.Model {
	if r == nil {
		return nil
	}
	if r.kind == rowArgs {
		return m.argInput(r)
	}
	if r.kind != rowOpt {
		return nil
	}
	o := &m.spec.Options[r.opt]
	switch o.Kind {
	case manpage.KindString, manpage.KindNumber, manpage.KindPath:
		return m.inputs[r.opt]
	case manpage.KindChoice:
		if m.customEdit == r.opt {
			return m.inputs[r.opt]
		}
	}
	return nil
}

func (m *Model) focusCurrent() tea.Cmd {
	m.blurInputs()
	if m.lib || m.naming {
		m.filter.Blur()
		return nil
	}
	if m.filtering {
		return m.filter.Focus()
	}
	m.filter.Blur()
	if ti := m.textInputFor(m.curRow()); ti != nil {
		ti.CursorEnd()
		return ti.Focus()
	}
	return nil
}

func (m *Model) move(delta int) tea.Cmd {
	if len(m.vis) == 0 {
		return nil
	}
	m.commitCustom()
	n := m.cursor
	for {
		n += delta
		if n < 0 || n >= len(m.vis) {
			break
		}
		if focusable(m.rows[m.vis[n]]) {
			m.cursor = n
			break
		}
	}
	m.dropdown = false
	m.helpScroll = 0
	return m.focusCurrent()
}

// toFirstOption puts the cursor on the first option row, or the first
// argument field when the command has no options.
func (m *Model) toFirstOption() tea.Cmd {
	for k, ri := range m.vis {
		if m.rows[ri].kind == rowOpt {
			m.cursor = k
			return m.move(0)
		}
	}
	m.cursor = 0
	return m.move(0)
}

func (m *Model) moveTo(n int) tea.Cmd {
	if n < 0 {
		n = 0
	}
	if n >= len(m.vis) {
		n = len(m.vis) - 1
	}
	// settle on a focusable row
	for k := n; k < len(m.vis); k++ {
		if focusable(m.rows[m.vis[k]]) {
			m.cursor = k
			return m.move(0)
		}
	}
	for k := n; k >= 0; k-- {
		if focusable(m.rows[m.vis[k]]) {
			m.cursor = k
			return m.move(0)
		}
	}
	return nil
}

func (m *Model) commitCustom() {
	if m.customEdit >= 0 {
		i := m.customEdit
		m.values[i].Text = strings.TrimSpace(m.inputs[i].Value())
		m.values[i].On = m.values[i].Text != ""
		m.customEdit = -1
	}
}

// syncText copies text input values into option values.
func (m *Model) syncText(i int) {
	ti := m.inputs[i]
	if ti == nil {
		return
	}
	v := ti.Value()
	m.values[i].Text = v
	m.values[i].On = strings.TrimSpace(v) != ""
	if m.values[i].On {
		m.applyConflicts(i)
	}
}

// setOn turns an option on or off, honoring radio groups and conflicts.
func (m *Model) setOn(i int, on bool) tea.Cmd {
	v := &m.values[i]
	v.On = on
	if on && v.Count == 0 {
		v.Count = 1
	}
	if !on {
		v.Count = 0
	}
	if !on {
		return nil
	}
	if g := m.spec.GroupOf(i); g >= 0 {
		for _, mi := range m.spec.Groups[g].Members {
			if mi != i {
				m.values[mi].On = false
				m.values[mi].Count = 0
			}
		}
	}
	return m.applyConflicts(i)
}

func (m *Model) applyConflicts(i int) tea.Cmd {
	o := &m.spec.Options[i]
	var off []string
	for _, name := range o.Conflicts {
		for j := range m.spec.Options {
			if j != i && m.spec.Options[j].HasName(name) && m.values[j].On {
				m.values[j] = cmdline.Value{}
				if ti := m.inputs[j]; ti != nil {
					ti.SetValue("")
				}
				off = append(off, name)
			}
		}
	}
	if len(off) > 0 {
		return m.setStatus(fmt.Sprintf("Turned off %s (conflicts with %s)", strings.Join(off, ", "), o.Names[0]), false)
	}
	return nil
}

func (m *Model) tokens() []cmdline.Token {
	if m.spec == nil {
		return nil
	}
	return cmdline.Build(m.spec, m.values, m.argsText(), m.preferLong)
}

// riskyInUse lists the set options that delete or overwrite data.
func (m *Model) riskyInUse() []string {
	var names []string
	for i, v := range m.values {
		if o := &m.spec.Options[i]; v.On && o.Danger != "" {
			names = append(names, o.Names[0])
		}
	}
	return names
}

func (m *Model) accept() tea.Cmd {
	m.commitCustom()
	if missing := m.missingArgs(); len(missing) > 0 && !m.warnedMissing {
		m.warnedMissing = true
		return m.setStatus(strings.Join(missing, ", ")+" looks required. Fill it in, or press ⏎ again to run anyway.", true)
	}
	if m.cfg.Confirm && m.confirming == nil {
		if risky := m.riskyInUse(); len(risky) > 0 {
			m.confirming = risky
			m.dropdown, m.filtering = false, false
			return m.focusCurrent()
		}
	}
	m.confirming = nil
	m.result = Result{Command: cmdline.Render(m.tokens()), Accepted: true, Key: m.spec.Command}
	return tea.Quit
}

func (m *Model) updateForm(k tea.KeyMsg) tea.Cmd {
	key := k.String()
	if key != "enter" {
		m.warnedMissing = false
	}
	if m.confirming != nil {
		if key == "y" || key == "Y" {
			return m.accept()
		}
		m.confirming = nil
		return m.setStatus("Not run. Adjust the options, or press ⏎ again.", false)
	}
	if m.naming {
		return m.updateNaming(k)
	}
	if m.lib {
		return m.updateLibrary(k)
	}
	r := m.curRow()

	// Dropdown open: it owns the keyboard.
	if m.dropdown && r != nil && r.kind == rowOpt {
		o := &m.spec.Options[r.opt]
		n := len(o.Choices) + 2 // (not set) + choices + custom
		switch key {
		case "up", "k", "ctrl+p", "shift+tab":
			m.ddSel = (m.ddSel - 1 + n) % n
		case "down", "j", "ctrl+n", "tab":
			m.ddSel = (m.ddSel + 1) % n
		case "home", "g":
			m.ddSel = 0
		case "end", "G":
			m.ddSel = n - 1
		case "enter", " ", "right", "l":
			m.dropdown = false
			return m.pickChoice(r.opt, m.ddSel)
		case "esc", "left", "h":
			m.dropdown = false
		}
		return nil
	}

	// Filter input focused.
	if m.filtering {
		switch key {
		case "esc":
			m.filter.SetValue("")
			m.filtering = false
			m.applyFilter()
			return m.focusCurrent()
		case "enter", "down", "tab":
			m.filtering = false
			if m.curRow() != nil && m.curRow().kind == rowArgs {
				m.move(1)
			}
			return m.focusCurrent()
		case "up":
			m.filtering = false
			return m.focusCurrent()
		}
		var cmd tea.Cmd
		m.filter, cmd = m.filter.Update(k)
		m.applyFilter()
		return cmd
	}

	ti := m.textInputFor(r)

	// Global keys.
	switch key {
	case "enter":
		if m.customEdit >= 0 {
			m.commitCustom()
			return m.focusCurrent()
		}
		return m.accept()
	case "esc":
		if m.customEdit >= 0 {
			m.inputs[m.customEdit].SetValue(m.values[m.customEdit].Text)
			m.customEdit = -1
			return m.focusCurrent()
		}
		if m.filter.Value() != "" {
			m.filter.SetValue("")
			m.applyFilter()
			return m.focusCurrent()
		}
		if m.fromSubs {
			return m.openSubs()
		}
		m.result = Result{}
		return tea.Quit
	case "ctrl+f":
		return m.startFilter("")
	case "ctrl+l":
		return m.openLibrary()
	case "ctrl+t":
		return m.startNaming()
	case "ctrl+o", "f1":
		m.openManual()
		return nil
	case "ctrl+y":
		return m.copy()
	case "ctrl+s":
		m.preferLong = !m.preferLong
		if m.preferLong {
			return m.setStatus("Using long option names", false)
		}
		return m.setStatus("Using short option names", false)
	case "ctrl+r":
		for i := range m.values {
			m.values[i] = cmdline.Value{}
		}
		for _, t := range m.inputs {
			t.SetValue("")
		}
		m.args.SetValue("")
		for _, t := range m.argInputs {
			t.SetValue("")
		}
		return m.setStatus("Cleared all options", false)
	case "up", "ctrl+p", "shift+tab":
		return m.move(-1)
	case "down", "ctrl+n":
		return m.move(1)
	case "tab":
		a, isArg := m.argSpec(r)
		pathArg := r.kind == rowArgs && (!isArg || a.Path)
		if ti != nil && (pathArg || (r.kind == rowOpt && m.spec.Options[r.opt].Kind == manpage.KindPath)) {
			if done, cmd := m.complete(ti, r.kind == rowArgs); done {
				return cmd
			}
		}
		return m.move(1)
	case "pgdown":
		return m.moveTo(m.cursor + m.listHeight() - 1)
	case "pgup":
		return m.moveTo(m.cursor - m.listHeight() + 1)
	case "shift+down", "alt+down":
		m.helpScroll++
		return nil
	case "shift+up", "alt+up":
		if m.helpScroll > 0 {
			m.helpScroll--
		}
		return nil
	}

	if ti != nil {
		if r.kind == rowArgs {
			var cmd tea.Cmd
			*ti, cmd = ti.Update(k)
			return cmd
		}
		o := &m.spec.Options[r.opt]
		if o.Kind == manpage.KindNumber && k.Type == tea.KeyRunes {
			switch key {
			case "+", "=":
				return m.bump(r.opt, 1)
			case "-", "_":
				return m.bump(r.opt, -1)
			}
			for _, c := range k.Runes {
				if (c < '0' || c > '9') && c != '.' {
					return m.setStatus(o.Names[0]+" expects a number", true)
				}
			}
		}
		var cmd tea.Cmd
		*ti, cmd = ti.Update(k)
		if m.customEdit < 0 {
			m.syncText(r.opt)
		}
		return cmd
	}

	if r == nil {
		return nil
	}
	if key == "home" {
		return m.moveTo(0)
	}
	if key == "end" {
		return m.moveTo(len(m.vis) - 1)
	}
	switch key {
	case "j":
		return m.move(1)
	case "k":
		return m.move(-1)
	case "/":
		return m.startFilter("")
	case "?":
		m.openManual()
		return nil
	}
	if r.kind != rowOpt {
		return nil
	}
	i := r.opt
	o := &m.spec.Options[i]
	v := &m.values[i]

	switch o.Kind {
	case manpage.KindFlag:
		switch key {
		case " ", "x":
			return m.setOn(i, !v.On)
		case "right", "l", "+":
			if v.On && o.Repeatable {
				v.Count++
				return nil
			}
			return m.setOn(i, true)
		case "left", "h", "-":
			if v.On && o.Repeatable && v.Count > 1 {
				v.Count--
				return nil
			}
			return m.setOn(i, false)
		case "backspace", "delete":
			return m.setOn(i, false)
		}
	case manpage.KindChoice:
		n := len(o.Choices)
		cur := indexOf(o.Choices, v.Text)
		switch key {
		case " ":
			m.dropdown = true
			m.ddSel = cur + 1
			if cur < 0 && v.Text != "" {
				m.ddSel = n + 1
			}
			return nil
		case "right", "l", "+":
			return m.pickChoice(i, (cur+1+1)%(n+1))
		case "left", "h", "-":
			return m.pickChoice(i, (cur+1-1+n+1)%(n+1))
		case "backspace", "delete", "x":
			return m.pickChoice(i, 0)
		}
	}
	// Any other printable key starts filtering.
	if k.Type == tea.KeyRunes && !k.Alt {
		return m.startFilter(string(k.Runes))
	}
	return nil
}

func indexOf(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

// pickChoice applies dropdown entry sel: 0 = unset, 1..n = choices,
// n+1 = custom value.
func (m *Model) pickChoice(i, sel int) tea.Cmd {
	o := &m.spec.Options[i]
	switch {
	case sel <= 0:
		m.values[i] = cmdline.Value{}
		m.inputs[i].SetValue("")
		return nil
	case sel <= len(o.Choices):
		m.values[i].Text = o.Choices[sel-1]
		m.inputs[i].SetValue(m.values[i].Text)
		return m.setOn(i, true)
	default:
		m.customEdit = i
		cur := m.values[i].Text
		if indexOf(o.Choices, cur) >= 0 {
			cur = "" // start fresh rather than editing a listed value
		}
		m.inputs[i].SetValue(cur)
		return m.focusCurrent()
	}
}

func (m *Model) bump(i, d int) tea.Cmd {
	ti := m.inputs[i]
	n, err := strconv.Atoi(strings.TrimSpace(ti.Value()))
	if err != nil {
		n = 0
	}
	n += d
	if n < 0 {
		n = 0
	}
	ti.SetValue(strconv.Itoa(n))
	ti.CursorEnd()
	m.syncText(i)
	return nil
}

func (m *Model) startFilter(initial string) tea.Cmd {
	m.commitCustom()
	m.filtering = true
	m.dropdown = false
	if initial != "" {
		m.filter.SetValue(m.filter.Value() + initial)
		m.filter.CursorEnd()
		m.applyFilter()
	}
	return m.focusCurrent()
}

func (m *Model) copy() tea.Cmd {
	s := cmdline.Render(m.tokens())
	for _, c := range [][]string{{"pbcopy"}, {"wl-copy"}, {"xclip", "-selection", "clipboard"}} {
		if _, err := exec.LookPath(c[0]); err == nil {
			cmd := exec.Command(c[0], c[1:]...)
			cmd.Stdin = strings.NewReader(s)
			if cmd.Run() == nil {
				return m.setStatus("Copied to clipboard", false)
			}
		}
	}
	if m.cfg.Output != nil {
		m.cfg.Output.Copy(s)
		return m.setStatus("Copied to clipboard (OSC 52)", false)
	}
	return m.setStatus("No clipboard available", true)
}

func (m *Model) forwardToFocused(msg tea.Msg) tea.Cmd {
	switch m.mode {
	case modePick:
		var cmd tea.Cmd
		m.pick, cmd = m.pick.Update(msg)
		return cmd
	case modeManual:
		if m.manFind {
			var cmd tea.Cmd
			m.manSearch, cmd = m.manSearch.Update(msg)
			return cmd
		}
	case modeSub:
		var cmd tea.Cmd
		m.subFilter, cmd = m.subFilter.Update(msg)
		return cmd
	case modeForm:
		if m.naming {
			var cmd tea.Cmd
			m.presetIn, cmd = m.presetIn.Update(msg)
			return cmd
		}
		if m.filtering {
			var cmd tea.Cmd
			m.filter, cmd = m.filter.Update(msg)
			return cmd
		}
		if ti := m.textInputFor(m.curRow()); ti != nil {
			var cmd tea.Cmd
			*ti, cmd = ti.Update(msg)
			return cmd
		}
	}
	return nil
}

func (m *Model) onMouse(msg tea.MouseMsg) tea.Cmd {
	switch m.mode {
	case modeManual:
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.scrollManual(-3)
		case tea.MouseButtonWheelDown:
			m.scrollManual(3)
		}
		return nil
	case modeSub:
		if n := len(m.subItems); n > 0 {
			switch msg.Button {
			case tea.MouseButtonWheelUp:
				m.subSel = max(0, m.subSel-1)
			case tea.MouseButtonWheelDown:
				m.subSel = min(n-1, m.subSel+1)
			}
		}
		return nil
	case modeForm:
	default:
		return nil
	}
	if m.dropdown || m.lib || m.naming {
		return nil
	}
	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.move(-1)
	case tea.MouseButtonWheelDown:
		return m.move(1)
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return nil
		}
		line := msg.Y - m.listTop
		if line < 0 || line >= len(m.lineRows) || m.lineRows[line] < 0 {
			return nil
		}
		target := m.lineRows[line]
		if !focusable(m.rows[m.vis[target]]) {
			return nil
		}
		same := target == m.cursor
		m.cursor = target
		cmd := m.move(0)
		if r := m.curRow(); same || (r.kind == rowOpt && m.spec.Options[r.opt].Kind == manpage.KindFlag) {
			r := m.curRow()
			if r.kind == rowOpt {
				o := &m.spec.Options[r.opt]
				switch o.Kind {
				case manpage.KindFlag:
					return tea.Batch(cmd, m.setOn(r.opt, !m.values[r.opt].On))
				case manpage.KindChoice:
					m.dropdown = true
					m.ddSel = indexOf(o.Choices, m.values[r.opt].Text) + 1
				}
			}
		}
		return cmd
	}
	return nil
}
