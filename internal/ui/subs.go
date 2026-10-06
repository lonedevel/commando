package ui

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lonedevel/commando/internal/manpage"
	"github.com/lonedevel/commando/internal/store"
)

// The subcommand browser lists a tool's commands (git add, git commit…)
// before any form, when the tool was opened without one.

const maxSubRecent = 5

type subKind int

const (
	subOwn    subKind = iota // the tool's own options
	subPreset                // a saved preset for the tool or a subcommand
	subRecent                // a recently run line
	subCmd                   // a subcommand
)

type subItem struct {
	kind  subKind
	sub   int // index into m.subs, for subCmd
	entry store.Entry
}

// subLevel is a list the browser can return to.
type subLevel struct {
	msg    loadedMsg
	subs   []manpage.Subcommand
	filter string
	sel    int
}

// openSubs shows the browser for the tool loaded in m.toolMsg.
func (m *Model) openSubs() tea.Cmd {
	m.mode = modeSub
	m.lib, m.naming, m.dropdown, m.filtering = false, false, false, false
	m.confirming = nil
	m.blurInputs()
	m.filterSubs()
	return m.subFilter.Focus()
}

func (m *Model) subTool() string { return m.toolMsg.spec.Command }

// filterSubs rebuilds the visible entries from the filter text.
func (m *Model) filterSubs() {
	q := strings.Fields(strings.ToLower(m.subFilter.Value()))
	matches := func(hay string) bool {
		hay = strings.ToLower(hay)
		for _, t := range q {
			if !strings.Contains(hay, t) {
				return false
			}
		}
		return true
	}
	m.subItems = m.subItems[:0]
	if len(q) == 0 {
		m.subItems = append(m.subItems, subItem{kind: subOwn})
	}
	if m.cfg.Store != nil {
		presets, recent := m.cfg.Store.Under(m.subTool(), maxSubRecent)
		for _, p := range presets {
			if matches(p.Name + " " + p.Line) {
				m.subItems = append(m.subItems, subItem{kind: subPreset, entry: p})
			}
		}
		for _, r := range recent {
			if matches(r.Line) {
				m.subItems = append(m.subItems, subItem{kind: subRecent, entry: r})
			}
		}
	}
	var cmds []subItem
	for i, s := range m.subs {
		if matches(s.Name + " " + strings.Join(s.Aliases, " ") + " " + s.Desc) {
			cmds = append(cmds, subItem{kind: subCmd, sub: i})
		}
	}
	if len(q) > 0 {
		// Commands whose name matches come before ones that only mention
		// the words in their description.
		sort.SliceStable(cmds, func(i, j int) bool {
			return m.subScore(cmds[i].sub, q[0]) > m.subScore(cmds[j].sub, q[0])
		})
	}
	m.subItems = append(m.subItems, cmds...)
	m.subSel = 0
	if len(q) == 0 && len(m.subItems) > 1 && m.subItems[1].kind != subCmd {
		m.subSel = 1 // the latest saved line: what you most likely want again
	}
	m.subOff = 0
}

func (m *Model) subScore(i int, t string) int {
	s := m.subs[i]
	best := 0
	for _, n := range append([]string{s.Name}, s.Aliases...) {
		switch {
		case n == t:
			return 3
		case strings.HasPrefix(n, t):
			best = max(best, 2)
		case strings.Contains(n, t):
			best = max(best, 1)
		}
	}
	return best
}

func (m *Model) updateSubs(k tea.KeyMsg) tea.Cmd {
	n := len(m.subItems)
	switch k.String() {
	case "shift+down":
		m.helpScroll++
		return nil
	case "shift+up":
		m.helpScroll = max(0, m.helpScroll-1)
		return nil
	}
	m.helpScroll = 0
	switch k.String() {
	case "esc":
		if m.subFilter.Value() != "" {
			m.subFilter.SetValue("")
			m.filterSubs()
			return nil
		}
		if n := len(m.subStack); n > 0 {
			// Back up one level: docker container → docker.
			up := m.subStack[n-1]
			m.subStack = m.subStack[:n-1]
			m.toolMsg, m.subs = up.msg, up.subs
			m.subFilter.SetValue(up.filter)
			m.subFilter.CursorEnd()
			m.filterSubs()
			m.subSel = min(up.sel, max(0, len(m.subItems)-1))
			return nil
		}
		m.result = Result{}
		return tea.Quit
	case "up", "ctrl+p", "shift+tab":
		if n > 0 {
			m.subSel = (m.subSel - 1 + n) % n
		}
		return nil
	case "down", "ctrl+n", "tab":
		if n > 0 {
			m.subSel = (m.subSel + 1) % n
		}
		return nil
	case "pgup":
		m.subSel = max(0, m.subSel-m.subListHeight())
		return nil
	case "pgdown":
		m.subSel = max(0, min(n-1, m.subSel+m.subListHeight()))
		return nil
	case "home":
		m.subSel = 0
		return nil
	case "end":
		m.subSel = max(0, n-1)
		return nil
	case "ctrl+o", "f1":
		m.manLines = colorizeManual(m.toolMsg.spec.Manual)
		m.manBack = modeSub
		m.manFind = false
		m.manOff = 0
		m.mode = modeManual
		m.subFilter.Blur()
		return nil
	case "enter":
		if m.subSel >= n {
			return nil
		}
		return m.pickSub(m.subItems[m.subSel])
	}
	var cmd tea.Cmd
	before := m.subFilter.Value()
	m.subFilter, cmd = m.subFilter.Update(k)
	if m.subFilter.Value() != before {
		m.filterSubs()
	}
	return cmd
}

// pickSub opens the form for the chosen entry.
func (m *Model) pickSub(it subItem) tea.Cmd {
	m.fromSubs = true
	m.subFilter.Blur()
	var line string
	switch it.kind {
	case subOwn:
		m.subPicked = ""
		return m.onLoaded(m.toolMsg)
	case subCmd:
		line = m.subTool() + " " + m.subs[it.sub].Name
		m.subPicked = line
	default:
		line = it.entry.Line
		m.subPicked = ""
	}
	m.cfg.Line = line
	m.mode = modeLoading
	m.loadWhat = line
	return tea.Batch(m.spin.Tick, m.load(line))
}

func (m *Model) subListHeight() int {
	_, h, _, _, _ := m.subLayout()
	return max(1, h-4)
}

// subLayout returns the browser's pane sizes.
func (m *Model) subLayout() (listW, listH, helpW, helpH int, side bool) {
	mid := m.h - 2
	if m.w >= 100 {
		listW = m.w * 62 / 100
		return listW, mid, m.w - listW, mid, true
	}
	if mid >= 18 {
		helpH = min(8, mid/3)
	}
	return m.w, mid - helpH, m.w, helpH, false
}

func (m *Model) viewSubs() string {
	listW, listH, helpW, helpH, side := m.subLayout()
	tool := m.toolMsg.spec
	title := " " + logo() + "  " + sCmd.Render(strings.Join(strings.Fields(tool.Command), " › "))
	if tool.Summary != "" {
		title += sDim.Render(" — " + tool.Summary)
	}
	info := sFaint.Render(itoa(len(m.subs)) + " commands ")
	if gap := m.w - ansi.StringWidth(title) - ansi.StringWidth(info); gap > 1 {
		title += strings.Repeat(" ", gap) + info
	}

	ncmd := 0
	for _, it := range m.subItems {
		if it.kind == subCmd {
			ncmd++
		}
	}
	listTitle := sHeader.Render("Commands") + sDim.Render(" · "+itoa(len(m.subs)))
	if m.subFilter.Value() != "" {
		listTitle = sHeader.Render("Commands") + sDim.Render(" · "+itoa(ncmd)+" of "+itoa(len(m.subs)))
	}
	listBox := box(listTitle, m.subsBody(listW-4, listH-2), listW, listH, cViolet)
	mid := listBox
	if helpH > 0 {
		helpBox := box(sHeader.Render("About"), m.scrollHelp(m.subHelp(helpW-4), helpH-2), helpW, helpH, cCyan)
		if side {
			mid = lipgloss.JoinHorizontal(lipgloss.Top, listBox, helpBox)
		} else {
			mid = listBox + "\n" + helpBox
		}
	}
	var foot string
	if m.status != "" {
		foot = " " + m.statusView()
	} else {
		esc := "quit"
		switch {
		case m.subFilter.Value() != "":
			esc = "clear"
		case len(m.subStack) > 0:
			esc = "back"
		}
		foot = " " + keyHelp([][2]string{{"type", "filter"}, {"↑↓", "choose"}, {"⏎", "open"}, {"^O", "manual"}, {"esc", esc}})
	}
	return fit(title, m.w) + "\n" + mid + "\n" + fit(foot, m.w)
}

// subsBody renders the filter line and the entries, grouped.
func (m *Model) subsBody(inner, height int) []string {
	m.subFilter.Width = max(8, inner-12)
	lines := []string{sKey.Render("Filter ") + m.subFilter.View(), ""}
	if len(m.subItems) == 0 {
		return append(lines, sDim.Render("No commands match."))
	}
	nameW := 0
	for _, s := range m.subs {
		nameW = max(nameW, ansi.StringWidth(s.Name))
	}
	nameW = min(nameW, max(8, min(16, inner/3)))
	filtering := m.subFilter.Value() != ""

	// Lay out every entry with its headings, then scroll to the selection.
	var body []string
	sel := 0
	heading := func(t string) {
		body = append(body, sHeader.Render("━━ "+t+" ")+sFaint.Render(strings.Repeat("─", max(0, inner-ansi.StringWidth(t)-4))))
	}
	lastGroup := "\x00"
	var lastKind subKind = -1
	for i, it := range m.subItems {
		on := i == m.subSel
		marker := "  "
		if on {
			marker = sCursor.Render("❯ ")
		}
		var text string
		switch it.kind {
		case subOwn:
			own := m.subTool()
			if pad := nameW - ansi.StringWidth(own); pad > 0 {
				own += strings.Repeat(" ", pad)
			}
			name := sBold.Render(own)
			if on {
				name = sCursor.Render(own)
			}
			text = name + "  " + sDim.Render("its own options, without a command")
		case subPreset, subRecent:
			if it.kind != lastKind {
				heading("Presets & recent")
			}
			if it.kind == subPreset {
				label := "★ " + it.entry.Name
				if on {
					text = sCursor.Render(label)
				} else {
					text = sBold.Render(label)
				}
				text += "  " + m.subLine(it.entry.Line)
			} else {
				text = sFaint.Render(fit(ago(it.entry.Used), 9)) + " " + m.subLine(it.entry.Line)
			}
		case subCmd:
			s := m.subs[it.sub]
			if !filtering && s.Group != lastGroup && (s.Group != "" || lastKind != subCmd) {
				g := s.Group
				if g == "" {
					g = "Commands"
				}
				heading(g)
			}
			lastGroup = s.Group
			name := sNameB.Render(fit(s.Name, nameW))
			if on {
				name = sCursor.Render(fit(s.Name, nameW))
			}
			desc := s.Desc
			if len(s.Aliases) > 0 {
				desc += sFaint.Render("  (" + strings.Join(s.Aliases, ", ") + ")")
			}
			text = name + "  " + sText.Render(desc)
		}
		if it.kind == subPreset {
			lastKind = subRecent // one heading for presets and recent together
		} else {
			lastKind = it.kind
		}
		if on {
			sel = len(body)
		}
		body = append(body, marker+fit(text, inner-2))
	}
	rows := height - len(lines)
	if rows <= 0 {
		return lines[:max(0, height)]
	}
	if sel < m.subOff+1 {
		m.subOff = max(0, sel-1) // keep the heading above the selection in view
	}
	if sel >= m.subOff+rows {
		m.subOff = sel - rows + 1
	}
	m.subOff = max(0, min(m.subOff, len(body)-rows))
	end := min(len(body), m.subOff+rows)
	return append(lines, body[m.subOff:end]...)
}

// subHelp describes the selected entry.
func (m *Model) subHelp(w int) []string {
	if m.subSel >= len(m.subItems) {
		return nil
	}
	it := m.subItems[m.subSel]
	tool := m.toolMsg.spec
	var lines []string
	switch it.kind {
	case subOwn:
		head := sCmd.Render(tool.Command)
		if tool.Summary != "" {
			head += sDim.Render(" — " + tool.Summary)
		}
		lines = append(lines, head, "")
		if d := firstPara(tool.Description); d != "" {
			for _, l := range wrap(d, w) {
				lines = append(lines, sText.Render(l))
			}
			lines = append(lines, "")
		}
		lines = append(lines, dimWrap("⏎ opens the form for "+tool.Command+"'s own options. Pick a command below to build one of those instead.", w)...)
	case subPreset, subRecent:
		if it.kind == subPreset {
			lines = append(lines, sBold.Render("★ "+it.entry.Name), sDim.Render("Preset, saved "+ago(it.entry.Used)), "")
		} else {
			lines = append(lines, sBold.Render("Recent command"), sDim.Render("Run "+ago(it.entry.Used)), "")
		}
		for i, l := range wrap(it.entry.Line, w) {
			if i == 0 {
				l = m.subLine(l)
			} else {
				l = m.styledLine(l, false)
			}
			lines = append(lines, l)
		}
		lines = append(lines, "")
		lines = append(lines, dimWrap("⏎ loads it into its form so you can adjust it before running.", w)...)
	case subCmd:
		s := m.subs[it.sub]
		lines = append(lines, sCmd.Render(tool.Command+" "+s.Name))
		if s.Group != "" {
			lines = append(lines, sDim.Render(s.Group))
		}
		lines = append(lines, "")
		if s.Desc != "" {
			for _, l := range wrap(s.Desc+".", w) {
				lines = append(lines, sText.Render(l))
			}
			lines = append(lines, "")
		}
		if len(s.Aliases) > 0 {
			lines = append(lines, sDim.Render("Also: ")+sNameB.Render(strings.Join(s.Aliases, ", ")), "")
		}
		lines = append(lines, dimWrap("⏎ opens the form for "+tool.Command+" "+s.Name+".", w)...)
	}
	return lines
}

// subLine highlights a saved line, coloring the tool and its command.
func (m *Model) subLine(line string) string {
	words := strings.Fields(line)
	// The words naming the tool ("docker container"), then a command of it.
	nc := 0
	for _, t := range strings.Fields(m.subTool()) {
		if nc < len(words) && words[nc] == t {
			nc++
		}
	}
	for i, w := range words {
		switch {
		case i < nc, i == nc && m.isSub(w):
			words[i] = sCmd.Render(w)
		case strings.HasPrefix(w, "-"):
			words[i] = sNameB.Render(w)
		default:
			words[i] = sArg.Render(w)
		}
	}
	return strings.Join(words, " ")
}

func (m *Model) isSub(name string) bool {
	for _, s := range m.subs {
		if s.Name == name {
			return true
		}
	}
	return false
}
