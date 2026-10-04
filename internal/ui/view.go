package ui

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lonedevel/commando/internal/cmdline"
	"github.com/lonedevel/commando/internal/manpage"
)

var (
	gradFrom = [3]int{0xF4, 0x72, 0xB6} // pink
	gradTo   = [3]int{0x22, 0xD3, 0xEE} // cyan
)

func itoa(n int) string { return strconv.Itoa(n) }

// View implements tea.Model.
func (m *Model) View() string {
	if m.w == 0 || m.h == 0 {
		return ""
	}
	switch m.mode {
	case modePick:
		return m.viewPick()
	case modeLoading:
		return m.center(m.spin.View() + " " + sText.Render("Reading the manual for ") + sCmd.Render(m.loadWhat) + sText.Render("…"))
	case modeManual:
		return m.viewManual()
	}
	return m.viewForm()
}

func (m *Model) center(s string) string {
	return lipgloss.Place(m.w, m.h, lipgloss.Center, lipgloss.Center, s)
}

func logo() string { return gradient("◆ commando", gradFrom, gradTo) }

func (m *Model) viewPick() string {
	w := min(72, m.w-4)
	var lines []string
	lines = append(lines, logo()+sDim.Render("  build a command line from its manual"), "")
	lines = append(lines, sBold.Render("Which command would you like to run?"), "")
	m.pick.Width = w - 8
	lines = append(lines, m.pick.View())
	if len(m.sugg) > 0 {
		lines = append(lines, "")
		for i, s := range m.sugg {
			if i == m.suggSel {
				lines = append(lines, sCursor.Render("  ▸ ")+sCmd.Render(s))
			} else {
				lines = append(lines, "    "+sText.Render(s))
			}
		}
	}
	if m.pickErr != "" {
		lines = append(lines, "", sErr.Render("✗ ")+sText.Render(m.pickErr))
	}
	lines = append(lines, "", keyHelp([][2]string{{"⏎", "open"}, {"tab", "complete"}, {"↑↓", "choose"}, {"esc", "quit"}}))
	card := box(sDim.Render("start"), lines, w, len(lines)+2, cViolet)
	return m.center(card)
}

func keyHelp(pairs [][2]string) string {
	parts := make([]string, len(pairs))
	for i, p := range pairs {
		parts[i] = sKey.Render(p[0]) + " " + sDim.Render(p[1])
	}
	return strings.Join(parts, sFaint.Render(" • "))
}

func (m *Model) statusView() string {
	if m.statusErr {
		return sErr.Render("✗ ") + sText.Render(m.status)
	}
	return sOK.Render("✓ ") + sText.Render(m.status)
}

// layout returns the geometry of the form panes.
func (m *Model) layout() (listW, listH, helpW, helpH, cmdH int, side bool) {
	cmdLines := len(m.commandLines(m.w - 4))
	cmdH = cmdLines + 2
	mid := m.h - 1 - cmdH - 1
	if m.w >= 100 {
		listW = m.w * 62 / 100
		return listW, mid, m.w - listW, mid, cmdH, true
	}
	helpH = 0
	if mid >= 18 {
		helpH = min(10, mid/3)
	}
	return m.w, mid - helpH, m.w, helpH, cmdH, false
}

func (m *Model) listHeight() int {
	_, lh, _, _, _, _ := m.layout()
	return max(1, lh-3)
}

func (m *Model) resizeInputs() {
	if m.spec == nil || m.w == 0 {
		return
	}
	listW, _, _, _, _, _ := m.layout()
	_, nameW, widgetW := m.columns(listW - 4)
	for _, ti := range m.inputs {
		ti.Width = max(4, widgetW-3)
	}
	m.args.Width = max(8, nameW+widgetW-2)
	m.filter.Width = max(8, listW-10)
}

// columns splits the list's inner width into label, names and widget columns.
func (m *Model) columns(inner int) (labelW, nameW, widgetW int) {
	nameW = 0
	hasValue := false
	for _, o := range m.spec.Options {
		nameW = max(nameW, ansi.StringWidth(strings.Join(o.Names, ", ")))
		if o.Kind != manpage.KindFlag {
			hasValue = true
		}
	}
	nameW = min(nameW, max(10, min(22, inner/4)))
	if hasValue {
		widgetW = min(24, max(12, inner/4))
	}
	labelW = inner - 6 - nameW - 1 - widgetW - 1
	if labelW < 16 {
		// narrow: give the label room, shrink names
		nameW = max(6, nameW-(16-labelW))
		labelW = inner - 6 - nameW - 1 - widgetW - 1
	}
	return labelW, nameW, widgetW
}

func (m *Model) viewForm() string {
	listW, listH, helpW, helpH, cmdH, side := m.layout()

	// Title bar.
	title := " " + logo() + "  " + sCmd.Render(m.spec.Command)
	if m.spec.Summary != "" {
		title += sDim.Render(" — " + m.spec.Summary)
	}
	info := sFaint.Render(m.loadInfo + " ")
	gap := m.w - ansi.StringWidth(title) - ansi.StringWidth(info)
	if gap > 1 {
		title += strings.Repeat(" ", gap) + info
	}
	title = fit(title, m.w)

	set := 0
	for _, v := range m.values {
		if v.On {
			set++
		}
	}
	nopts := 0
	for _, ri := range m.vis {
		if m.rows[ri].kind == rowOpt {
			nopts++
		}
	}
	listTitle := sHeader.Render("Options") + sDim.Render(" · "+itoa(len(m.spec.Options))+" total")
	if m.filter.Value() != "" {
		listTitle = sHeader.Render("Options") + sDim.Render(" · "+itoa(nopts)+" of "+itoa(len(m.spec.Options)))
	}
	if set > 0 {
		listTitle += sDim.Render(" · ") + sOn.Render(itoa(set)+" set")
	}
	listBox := box(listTitle, m.listBody(listW-4, listH-2), listW, listH, cViolet)

	var mid string
	if helpH > 0 {
		helpBox := box(sHeader.Render(m.helpTitle()), m.helpBody(helpW-4, helpH-2), helpW, helpH, cCyan)
		if side {
			mid = lipgloss.JoinHorizontal(lipgloss.Top, listBox, helpBox)
		} else {
			mid = listBox + "\n" + helpBox
		}
	} else {
		mid = listBox
	}

	cmdBox := box(sHeader.Render("Command")+sDim.Render(" · ⏎ to run"), m.commandLines(m.w-4), m.w, cmdH, cPink)

	var foot string
	if m.status != "" {
		foot = " " + m.statusView()
	} else {
		foot = " " + m.footerKeys()
	}
	return title + "\n" + mid + "\n" + cmdBox + "\n" + fit(foot, m.w)
}

func (m *Model) helpTitle() string {
	r := m.curRow()
	if r == nil || r.kind == rowArgs {
		return "About " + m.spec.Command
	}
	return "Help"
}

func (m *Model) footerKeys() string {
	var pairs [][2]string
	if m.filtering {
		pairs = [][2]string{{"type", "filter"}, {"↓/⏎", "to list"}, {"esc", "clear"}}
		return keyHelp(pairs)
	}
	if m.dropdown {
		return keyHelp([][2]string{{"↑↓", "choose"}, {"⏎/space", "select"}, {"esc", "close"}})
	}
	if m.customEdit >= 0 {
		return keyHelp([][2]string{{"type", "custom value"}, {"⏎", "done"}, {"esc", "cancel"}})
	}
	r := m.curRow()
	if r != nil {
		switch {
		case r.kind == rowArgs:
			pairs = append(pairs, [2]string{"type", "arguments"}, [2]string{"tab", "complete path"})
		case r.kind == rowOpt:
			switch m.spec.Options[r.opt].Kind {
			case manpage.KindFlag:
				pairs = append(pairs, [2]string{"space", "toggle"})
				if m.spec.Options[r.opt].Repeatable {
					pairs = append(pairs, [2]string{"←→", "count"})
				}
			case manpage.KindChoice:
				pairs = append(pairs, [2]string{"space", "open"}, [2]string{"←→", "cycle"})
			case manpage.KindNumber:
				pairs = append(pairs, [2]string{"0-9", "edit"}, [2]string{"+/-", "adjust"})
			case manpage.KindPath:
				pairs = append(pairs, [2]string{"type", "edit"}, [2]string{"tab", "complete"})
			default:
				pairs = append(pairs, [2]string{"type", "edit"})
			}
		}
	}
	pairs = append(pairs, [2]string{"↑↓", "move"}, [2]string{"⏎", "run"}, [2]string{"^F", "filter"},
		[2]string{"^O", "manual"}, [2]string{"^Y", "copy"}, [2]string{"^S", "long/short"}, [2]string{"esc", "quit"})
	return keyHelp(pairs)
}

// commandLines renders the highlighted command, wrapped to w-4 cells.
func (m *Model) commandLines(w int) []string {
	inner := max(10, w-4)
	toks := m.tokens()
	prompt := sCursor.Render("❯ ")
	var lines []string
	cur, curW := prompt, 2
	for i, t := range toks {
		var st lipgloss.Style
		switch t.Kind {
		case cmdline.TokCommand:
			st = sCmd
		case cmdline.TokFlag:
			st = sNameB
		case cmdline.TokValue:
			st = sValue
		default:
			st = sArg
		}
		text := t.Text
		tw := ansi.StringWidth(text)
		sep := 1
		if i == 0 || t.Attach {
			sep = 0
		}
		if curW+sep+tw > inner && curW > 2 && !t.Attach {
			lines = append(lines, cur)
			cur, curW, sep = "  ", 2, 0
		}
		if sep == 1 {
			cur += " "
		}
		cur += st.Render(text)
		curW += sep + tw
	}
	lines = append(lines, cur)
	if len(lines) > 3 {
		lines = append(lines[:2], ansi.Truncate(lines[2], inner-1, "")+sDim.Render("…"))
	}
	return lines
}

// listBody renders the filter line and the visible option rows.
func (m *Model) listBody(inner, height int) []string {
	lines := make([]string, 0, height)
	// Filter line.
	icon := sHeader.Render("⌕ ")
	if m.filtering {
		lines = append(lines, icon+m.filter.View())
	} else if m.filter.Value() != "" {
		lines = append(lines, icon+sValue.Render(m.filter.Value())+sFaint.Render("  (esc clears)"))
	} else {
		lines = append(lines, icon+sFaint.Render("press / or ^F to filter options"))
	}
	rowsH := height - 1
	if rowsH <= 0 {
		return lines
	}
	labelW, nameW, widgetW := m.columns(inner)

	ddN := 0
	if m.dropdown {
		ddN = m.dropdownLen()
	}
	// Scroll so the cursor (and its dropdown) stay in view.
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor+ddN >= m.offset+rowsH {
		m.offset = m.cursor + ddN - rowsH + 1
	}
	// Reveal the section header directly above the cursor.
	if m.offset > 0 && m.offset == m.cursor && m.rows[m.vis[m.cursor-1]].kind != rowOpt {
		m.offset--
	}
	total := len(m.vis) + ddN
	if m.offset > max(0, total-rowsH) {
		m.offset = max(0, total-rowsH)
	}

	m.lineRows = m.lineRows[:0]
	m.lineRows = append(m.lineRows, -1) // filter line
	m.listTop = 3 - 1                   // title + top border; filter line is lineRows[0]
	for line := m.offset; line < total && len(lines) < height; line++ {
		var k, dd int = line, -1
		if m.dropdown && line > m.cursor {
			if line <= m.cursor+ddN {
				dd = line - m.cursor - 1
				k = m.cursor
			} else {
				k = line - ddN
			}
		}
		if dd >= 0 {
			lines = append(lines, m.dropdownLine(dd, 6+2))
			m.lineRows = append(m.lineRows, -1)
			continue
		}
		lines = append(lines, m.renderRow(k, labelW, nameW, widgetW, inner))
		m.lineRows = append(m.lineRows, k)
	}
	// Scroll indicator.
	if total > rowsH && len(lines) > 1 {
		pct := 100 * m.offset / max(1, total-rowsH)
		lines[0] = fit(lines[0], inner-6) + sFaint.Render(lipgloss.PlaceHorizontal(6, lipgloss.Right, itoa(pct)+"%"))
	}
	return lines
}

func (m *Model) renderRow(k, labelW, nameW, widgetW, inner int) string {
	r := m.rows[m.vis[k]]
	sel := k == m.cursor
	marker := "  "
	if sel {
		marker = sCursor.Render("❯ ")
	}
	switch r.kind {
	case rowHeader:
		t := " " + r.text + " "
		return sHeader.Render("━━"+t) + sFaint.Render(strings.Repeat("─", max(0, inner-ansi.StringWidth(t)-2)))
	case rowGroup:
		return "  " + sGroup.Render("◇ "+r.text) + sDim.Render(" · choose one")
	case rowArgs:
		label := sArg.Bold(true).Render("Arguments")
		if sel {
			label = sCursor.Render("Arguments")
		}
		wv := nameW + 1 + widgetW
		if wv < 10 {
			wv = inner - 6 - labelW - 1
		}
		var field string
		if sel {
			field = m.args.View()
		} else if v := m.args.Value(); v != "" {
			field = sArg.Render(v)
		} else {
			field = sFaint.Render(strings.ToLower(argHint(m.spec)))
		}
		return marker + sArg.Render("»") + "   " + fit(label, labelW) + " " + bracket(field, wv, sel)
	}

	i := r.opt
	o := &m.spec.Options[i]
	v := m.values[i]
	var ctrl string
	switch {
	case r.group >= 0 && v.On:
		ctrl = sOn.Render("(●)")
	case r.group >= 0:
		ctrl = sFaint.Render("( )")
	case v.On && v.Count > 1:
		ctrl = sOn.Render("[" + itoa(v.Count) + "]")
	case v.On:
		ctrl = sOn.Render("[✓]")
	default:
		ctrl = sFaint.Render("[ ]")
	}
	ctrl = fit(ctrl, 4)

	lst := sText
	if v.On {
		lst = sBold
	}
	if sel {
		lst = sCursor
	}
	names := fit(sName.Render(strings.Join(o.Names, ", ")), nameW)
	if o.Kind == manpage.KindFlag || widgetW == 0 {
		// Flags have no value widget: the label takes its space.
		w := labelW
		if widgetW > 0 {
			w += widgetW + 1
		}
		return marker + ctrl + fit(lst.Render(o.Label), w) + " " + names
	}
	label := fit(lst.Render(o.Label), labelW)

	var widget string
	switch o.Kind {
	case manpage.KindChoice:
		var val string
		switch {
		case m.customEdit == i:
			val = m.inputs[i].View()
		case v.Text != "":
			val = sValue.Render(v.Text)
		default:
			val = sFaint.Render("—")
		}
		arrow := sFaint.Render("▾")
		if sel {
			arrow = sHeader.Render("▾")
		}
		widget = bracket(fit(val, widgetW-4)+" "+arrow, widgetW, sel)
	default:
		var val string
		switch {
		case sel:
			val = m.inputs[i].View()
		case v.Text != "":
			val = sValue.Render(v.Text)
		default:
			val = sFaint.Render(strings.ToLower(o.Arg))
		}
		widget = bracket(val, widgetW, sel)
	}
	return marker + ctrl + label + " " + widget + " " + names
}

var hintRe = regexp.MustCompile(`^<?[A-Za-z][\w.-]*>?(\.\.\.)?$`)

func argHint(s *manpage.Spec) string {
	syn := strings.SplitN(s.Synopsis, "\n", 2)[0]
	fields := strings.Fields(strings.NewReplacer("[", "", "]", "").Replace(syn))
	var hint []string
	for _, f := range fields[min(1, len(fields)):] {
		u := strings.ToUpper(f)
		if !hintRe.MatchString(f) || strings.HasPrefix(u, "OPTION") {
			continue
		}
		hint = append(hint, f)
	}
	if len(hint) > 3 {
		hint = hint[:3]
	}
	if len(hint) == 0 {
		return "arguments"
	}
	return strings.Join(hint, " ")
}

func bracket(content string, w int, focused bool) string {
	bs := sFaint
	if focused {
		bs = sHeader
	}
	return bs.Render("[") + fit(content, w-2) + bs.Render("]")
}

func (m *Model) dropdownLen() int {
	r := m.curRow()
	if r == nil || r.kind != rowOpt {
		return 0
	}
	return min(len(m.spec.Options[r.opt].Choices)+2, 9)
}

// dropdownLine renders entry dd (0-based visible line) of the open dropdown.
func (m *Model) dropdownLine(dd, indent int) string {
	r := m.curRow()
	o := &m.spec.Options[r.opt]
	vis := m.dropdownLen()
	start := 0
	if m.ddSel >= vis {
		start = m.ddSel - vis + 1
	}
	idx := start + dd
	var text string
	switch {
	case idx == 0:
		text = sFaint.Render("(not set)")
	case idx <= len(o.Choices):
		text = sValue.Render(o.Choices[idx-1])
		if o.Choices[idx-1] == m.values[r.opt].Text {
			text += sOn.Render(" ✓")
		}
	default:
		text = sDim.Render("✎ custom value…")
	}
	pre := strings.Repeat(" ", indent) + sHeader.Render("│ ")
	if idx == m.ddSel {
		return pre + sCursor.Render("▸ ") + text
	}
	return pre + "  " + text
}

// helpBody renders the help pane for the focused row.
func (m *Model) helpBody(w, h int) []string {
	var lines []string
	add := func(s ...string) { lines = append(lines, s...) }
	r := m.curRow()
	if r == nil || r.kind != rowOpt {
		if m.spec.Summary != "" {
			add(sCmd.Render(m.spec.Command)+sDim.Render(" — ")+sText.Render(m.spec.Summary), "")
		}
		if m.spec.Synopsis != "" {
			add(sHeader.Render("Synopsis"))
			for _, l := range strings.Split(m.spec.Synopsis, "\n") {
				for _, wl := range wrap(l, w-2) {
					add("  " + sName.Render(wl))
				}
			}
			add("")
		}
		if m.spec.Description != "" {
			add(sHeader.Render("Description"))
			for i, p := range strings.Split(m.spec.Description, "\n\n") {
				if i > 0 {
					add("")
				}
				for _, wl := range wrap(p, w) {
					add(sText.Render(wl))
				}
			}
			add("")
		}
		add(sDim.Render("Tip: Tab completes file paths here. Press ^O for the full manual."))
		return m.scrollHelp(lines, h)
	}

	o := &m.spec.Options[r.opt]
	head := sNameB.Render(strings.Join(o.Names, ", "))
	if o.Arg != "" {
		a := o.Arg
		if o.ArgOptional {
			a = "[" + a + "]"
		}
		head += " " + sValue.Render(a)
	}
	add(head)
	tags := badge(o.Kind.String(), kindColor[o.Kind.String()])
	if o.Repeatable {
		tags += " " + badge("repeatable", cPink)
	}
	if o.Section != "" && o.Section != "Description" && o.Section != "Options" {
		tags += " " + sDim.Render(o.Section)
	}
	add(tags)
	if g := m.spec.GroupOf(r.opt); g >= 0 {
		add(sGroup.Render("◇ "+m.spec.Groups[g].Label) + sDim.Render(" — only one of these can be chosen"))
	}
	if len(o.Choices) > 0 {
		vals := strings.Join(o.Choices, " · ")
		for i, wl := range wrap("Values: "+vals, w) {
			if i == 0 {
				wl = strings.TrimPrefix(wl, "Values: ")
				add(sDim.Render("Values: ") + sValue.Render(wl))
			} else {
				add(sValue.Render(wl))
			}
		}
	}
	if len(o.Conflicts) > 0 {
		add(sDim.Render("Conflicts with: ") + sName.Render(strings.Join(o.Conflicts, ", ")))
	}
	add("")
	for i, p := range strings.Split(o.Desc, "\n\n") {
		if i > 0 {
			add("")
		}
		pre := strings.HasPrefix(p, "  ")
		if !pre {
			p = capitalizeFirst(p)
		}
		for _, wl := range wrap(p, w) {
			if pre {
				add(sDim.Render(ansi.Truncate(wl, w, "…")))
			} else {
				add(sText.Render(wl))
			}
		}
	}
	if len(o.Notes) > 0 {
		add("", sHeader.Render("More from the manual"))
		for i, p := range o.Notes {
			if i > 0 {
				add("")
			}
			for _, wl := range wrap(p, w) {
				add(sDim.Render(wl))
			}
		}
	}
	return m.scrollHelp(lines, h)
}

func (m *Model) scrollHelp(lines []string, h int) []string {
	if len(lines) <= h {
		m.helpScroll = 0
		return lines
	}
	maxScroll := len(lines) - h + 1
	if m.helpScroll > maxScroll {
		m.helpScroll = maxScroll
	}
	out := append([]string{}, lines[m.helpScroll:min(len(lines), m.helpScroll+h-1)]...)
	more := len(lines) - (m.helpScroll + h - 1)
	if more > 0 {
		out = append(out, sFaint.Render("  ↓ "+itoa(more)+" more lines · shift+↓ to scroll · ^O manual"))
	} else {
		out = append(out, sFaint.Render("  ↑ shift+↑ to scroll back"))
	}
	return out
}

func capitalizeFirst(s string) string {
	for i, r := range s {
		return s[:i] + strings.ToUpper(string(r)) + s[i+len(string(r)):]
	}
	return s
}
