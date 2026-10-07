package ui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/lonedevel/commando/internal/cmdline"
	"github.com/lonedevel/commando/internal/store"
)

type libKind int

const (
	libBlank libKind = iota
	libPreset
	libRecent
	libExample // from the manual's EXAMPLES section
)

type libItem struct {
	kind     libKind
	entry    store.Entry
	copyOnly bool // an example the form can't hold exactly; ⏎ copies it instead
	opt      int  // for an example given under an option, its index; else -1
}

// loadLibrary rebuilds the presets, recent and examples list for the
// current command and returns how many saved entries (presets and recent
// commands) it holds.
func (m *Model) loadLibrary() int {
	m.libItems = []libItem{{kind: libBlank}}
	m.libSel = 0
	if m.spec == nil {
		return 0
	}
	saved := 0
	if m.cfg.Store != nil {
		presets, recent := m.cfg.Store.For(m.spec.Command)
		for _, p := range presets {
			m.libItems = append(m.libItems, libItem{kind: libPreset, entry: p})
		}
		for _, r := range recent {
			m.libItems = append(m.libItems, libItem{kind: libRecent, entry: r})
		}
		saved = len(presets) + len(recent)
	}
	seen := map[string]bool{}
	addExample := func(line, desc string, opt int) {
		if seen[line] || hasShellOperator(line) {
			return
		}
		seen[line] = true
		words := cmdline.Split(line)[len(strings.Fields(m.spec.Command)):]
		m.libItems = append(m.libItems, libItem{
			kind:     libExample,
			entry:    store.Entry{Name: desc, Line: line},
			copyOnly: !cmdline.Faithful(m.spec, words),
			opt:      opt,
		})
	}
	for _, e := range m.spec.Examples {
		addExample(e.Line, e.Desc, -1)
	}
	// Examples given under each option ("Example: curl --retry 7 …").
	for i := range m.spec.Options {
		o := &m.spec.Options[i]
		for _, line := range o.Examples {
			addExample(line, o.Names[len(o.Names)-1]+": "+strings.TrimSuffix(o.Label, "."), i)
		}
	}
	return saved
}

func (m *Model) openLibrary() tea.Cmd {
	m.commitCustom()
	m.dropdown = false
	m.filtering = false
	m.loadLibrary()
	if len(m.libItems) == 1 {
		if m.cfg.Store == nil {
			return m.setStatus("Presets and history are turned off, and the manual has no examples", true)
		}
		return m.setStatus("No presets or recent commands yet. ^T saves this form as a preset.", false)
	}
	m.lib = true
	m.libSel = 1
	return m.focusCurrent()
}

// openExamples shows the list at the manual's first example, or at the
// focused option's own example when it has one.
func (m *Model) openExamples() tea.Cmd {
	if !m.hasExamples() {
		return m.setStatus("The manual for "+m.spec.Command+" has no examples", false)
	}
	opt := -1
	if r := m.curRow(); r != nil && r.kind == rowOpt {
		opt = r.opt
	}
	cmd := m.openLibrary()
	first := -1
	for i, it := range m.libItems {
		if it.kind != libExample {
			continue
		}
		if first < 0 {
			first = i
		}
		if opt >= 0 && it.opt == opt {
			first = i
			break
		}
	}
	if first >= 0 {
		m.libSel = first
	}
	return cmd
}

// hasExamples reports whether the manual gives any examples.
func (m *Model) hasExamples() bool {
	if len(m.spec.Examples) > 0 {
		return true
	}
	for i := range m.spec.Options {
		if len(m.spec.Options[i].Examples) > 0 {
			return true
		}
	}
	return false
}

// hasShellOperator reports whether line pipes, chains or redirects, so the
// form can't hold it.
func hasShellOperator(line string) bool {
	return len(splitLine(line)) > 1 || strings.ContainsAny(line, "<>`")
}

func (m *Model) closeLibrary() tea.Cmd {
	m.lib = false
	return m.focusCurrent()
}

func (m *Model) updateLibrary(k tea.KeyMsg) tea.Cmd {
	n := len(m.libItems)
	switch k.String() {
	case "up", "k", "ctrl+p", "shift+tab":
		m.libSel = (m.libSel - 1 + n) % n
	case "down", "j", "ctrl+n", "tab":
		m.libSel = (m.libSel + 1) % n
	case "home", "g":
		m.libSel = 0
	case "end", "G":
		m.libSel = n - 1
	case "enter", " ":
		it := m.libItems[m.libSel]
		if it.copyOnly {
			return m.copyText(it.entry.Line, "Copied the example. It uses operators or an order the form can't keep, so it's copied as written")
		}
		m.lib = false
		if it.kind == libBlank {
			return m.focusCurrent()
		}
		m.applyLine(it.entry.Line)
		what := "recent command"
		switch it.kind {
		case libPreset:
			what = "preset “" + it.entry.Name + "”"
		case libExample:
			what = "example"
		}
		return tea.Batch(m.focusCurrent(), m.setStatus("Loaded "+what+". Adjust it, then ⏎ to run.", false))
	case "ctrl+y":
		if it := m.libItems[m.libSel]; it.kind != libBlank {
			return m.copyText(it.entry.Line, "Copied to clipboard")
		}
	case "d", "x", "delete", "backspace":
		return m.deleteLibraryItem()
	case "esc", "ctrl+l", "ctrl+x":
		return m.closeLibrary()
	case "ctrl+t":
		m.lib = false
		return m.startNaming()
	case "ctrl+o", "f1", "?":
		m.lib = false
		m.openManual()
	}
	return nil
}

func (m *Model) deleteLibraryItem() tea.Cmd {
	it := m.libItems[m.libSel]
	switch it.kind {
	case libPreset:
		m.cfg.Store.DeletePreset(m.spec.Command, it.entry.Name)
	case libRecent:
		m.cfg.Store.DeleteRecent(m.spec.Command, it.entry.Line)
	case libExample:
		return m.setStatus("Examples come from the manual, so they can't be deleted", false)
	default:
		return nil
	}
	err := m.cfg.Store.Save()
	sel := m.libSel
	if m.loadLibrary() == 0 {
		m.lib = false
		m.focusCurrent()
	}
	m.libSel = max(0, min(sel, len(m.libItems)-1))
	if err != nil {
		return m.setStatus("Could not save: "+err.Error(), true)
	}
	return m.setStatus("Deleted", false)
}

// applyLine resets the form to a saved command line.
func (m *Model) applyLine(line string) {
	words := cmdline.Split(line)
	// Skip the command itself ("ls", or "git commit").
	for _, w := range strings.Fields(m.spec.Command) {
		if len(words) > 0 && words[0].Value == w {
			words = words[1:]
		}
	}
	m.fill(words)
	m.filter.SetValue("")
	m.applyFilter()
	m.toFirstOption()
}

func (m *Model) startNaming() tea.Cmd {
	if m.cfg.Store == nil {
		return m.setStatus("Presets and history are turned off", true)
	}
	m.commitCustom()
	m.dropdown = false
	m.filtering = false
	m.naming = true
	m.presetIn.SetValue("")
	m.blurInputs()
	m.filter.Blur()
	return m.presetIn.Focus()
}

func (m *Model) updateNaming(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		m.naming = false
		m.presetIn.Blur()
		return m.focusCurrent()
	case "enter":
		name := strings.TrimSpace(m.presetIn.Value())
		if name == "" {
			return m.setStatus("Type a name for the preset", true)
		}
		m.naming = false
		m.presetIn.Blur()
		line := cmdline.Render(m.tokens())
		m.cfg.Store.SavePreset(m.spec.Command, name, line, time.Now())
		if err := m.cfg.Store.Save(); err != nil {
			return tea.Batch(m.focusCurrent(), m.setStatus("Could not save preset: "+err.Error(), true))
		}
		return tea.Batch(m.focusCurrent(), m.setStatus(fmt.Sprintf("Saved preset “%s”. ^L lists presets.", name), false))
	}
	var cmd tea.Cmd
	m.presetIn, cmd = m.presetIn.Update(k)
	return cmd
}

// libraryBody renders the presets & recent list in place of the options.
func (m *Model) libraryBody(inner, height int) []string {
	head := []string{sDim.Render("Start from a saved command, an example, or a blank form:"), ""}
	// Lay out every entry with its headings, then scroll to the selection.
	var lines []string
	selLine := 0
	lastKind := libBlank
	for i := 0; i < len(m.libItems); i++ {
		it := m.libItems[i]
		if it.kind != lastKind && i > 0 {
			title := "Presets"
			switch it.kind {
			case libRecent:
				title = "Recent"
			case libExample:
				title = "Examples from the manual"
			}
			lines = append(lines, sHeader.Render("━━ "+title+" ")+sFaint.Render(strings.Repeat("─", max(0, inner-len(title)-4))))
		}
		lastKind = it.kind
		sel := i == m.libSel
		if sel {
			selLine = len(lines)
		}
		marker := "  "
		if sel {
			marker = sCursor.Render("❯ ")
		}
		var text string
		switch it.kind {
		case libBlank:
			text = sText.Render("Blank form")
			if sel {
				text = sCursor.Render("Blank form")
			}
		case libPreset:
			name := sBold.Render("★ " + it.entry.Name)
			if sel {
				name = sCursor.Render("★ " + it.entry.Name)
			}
			text = name + "  " + m.styledLine(it.entry.Line, true)
		case libRecent:
			when := sFaint.Render(fit(ago(it.entry.Used), 9))
			text = when + " " + m.styledLine(it.entry.Line, true)
		case libExample:
			text = m.styledLine(it.entry.Line, true)
			if it.copyOnly {
				text = sFaint.Render("⧉ ") + text
			}
			if it.entry.Name != "" {
				text += sFaint.Render("  " + it.entry.Name)
			}
		}
		lines = append(lines, marker+fit(text, inner-2))
	}
	rows := height - len(head)
	if rows <= 0 {
		return head[:max(0, height)]
	}
	// Keep the selection in view, with its heading when there's room.
	start := 0
	if selLine >= rows {
		start = selLine - rows + 1
	}
	if start > 0 && selLine-1 >= 0 && selLine-1 < start+rows-1 {
		start = max(0, min(start, selLine-1))
	}
	end := min(len(lines), start+rows)
	return append(head, lines[start:end]...)
}

// styledLine highlights a saved command line like the command preview.
func (m *Model) styledLine(line string, withCmd bool) string {
	words := strings.Fields(line)
	nc := 0
	if withCmd {
		nc = len(strings.Fields(m.spec.Command))
	}
	for i, w := range words {
		switch {
		case i < nc:
			words[i] = sCmd.Render(w)
		case strings.HasPrefix(w, "-"):
			words[i] = sNameB.Render(w)
		default:
			words[i] = sArg.Render(w)
		}
	}
	return strings.Join(words, " ")
}

// libraryHelp explains the selected entry in the help pane.
func (m *Model) libraryHelp(w int) []string {
	it := m.libItems[m.libSel]
	var lines []string
	switch it.kind {
	case libBlank:
		lines = append(lines, sBold.Render("Blank form"), "",
			sText.Render("Start with no options set."))
	case libPreset:
		lines = append(lines, sBold.Render("★ "+it.entry.Name), sDim.Render("Preset, saved "+ago(it.entry.Used)), "")
	case libRecent:
		lines = append(lines, sBold.Render("Recent command"), sDim.Render("Run "+ago(it.entry.Used)), "")
	case libExample:
		lines = append(lines, sBold.Render("Example from the manual"), "")
		if it.entry.Name != "" {
			for _, l := range wrap(it.entry.Name, w) {
				lines = append(lines, sText.Render(l))
			}
			lines = append(lines, "")
		}
	}
	if it.kind == libExample {
		for i, wl := range wrap(it.entry.Line, w) {
			lines = append(lines, m.styledLine(wl, i == 0))
		}
		lines = append(lines, "")
		if it.copyOnly {
			lines = append(lines, dimWrap("⧉ The form can't hold this one exactly: it uses operators such as ! or ( ), repeats an option, or puts options after its arguments. ⏎ copies it to the clipboard as written.", w)...)
		} else {
			lines = append(lines, dimWrap("⏎ loads it into the form so you can adjust it before running.", w)...)
		}
	} else if it.kind != libBlank {
		for i, wl := range wrap(it.entry.Line, w) {
			lines = append(lines, m.styledLine(wl, i == 0))
		}
		lines = append(lines, "")
		lines = append(lines, dimWrap("⏎ loads it into the form so you can adjust it before running. d deletes it.", w)...)
	}
	if it.kind != libBlank {
		if parts := m.explainEntry(it.entry.Line); len(parts) > 0 {
			lines = append(lines, "", sHeader.Render("What it does"))
			lines = append(lines, explainParts(m.spec, parts, w, true)...)
		}
	}
	lines = append(lines, "")
	return append(lines, dimWrap("^T saves the current form as a preset.", w)...)
}

// explainEntry breaks a saved line for this command into its options and
// arguments.
func (m *Model) explainEntry(line string) []cmdline.Part {
	words := cmdline.Split(line)
	n := len(strings.Fields(m.spec.Command))
	if len(words) <= n {
		return nil
	}
	return cmdline.Explain(m.spec, words[n:])
}

func ago(t time.Time) string {
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	case d < 30*24*time.Hour:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
	return t.Format("Jan 2")
}

func dimWrap(s string, w int) []string {
	var out []string
	for _, l := range wrap(s, w) {
		out = append(out, sDim.Render(l))
	}
	return out
}

// libraryTitle names what the list holds.
func (m *Model) libraryTitle() string {
	saved, examples := false, false
	for _, it := range m.libItems {
		switch it.kind {
		case libPreset, libRecent:
			saved = true
		case libExample:
			examples = true
		}
	}
	switch {
	case saved && examples:
		return "Presets, recent & examples"
	case examples:
		return "Examples"
	}
	return "Presets & recent"
}
