package ui

import (
	"regexp"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/lonedevel/commando/internal/manpage"
)

var manOptRe = regexp.MustCompile(`^(\s+)(-[^\s].*?)(\s{2,}.*)?$`)

// colorizeManual styles headings, subsections and option tags.
func colorizeManual(text string) []string {
	raw := strings.Split(strings.TrimRight(text, "\n"), "\n")
	out := make([]string, len(raw))
	for i, l := range raw {
		ind := len(l) - len(strings.TrimLeft(l, " "))
		t := strings.TrimSpace(l)
		switch {
		case t == "":
			out[i] = ""
		case ind == 0:
			out[i] = sGroup.Render(l)
		case ind <= 4 && !strings.HasPrefix(t, "-"):
			out[i] = sHeader.Render(l)
		default:
			if m := manOptRe.FindStringSubmatch(l); m != nil && len(m[1]) <= 12 {
				out[i] = m[1] + sNameB.Render(m[2]) + sText.Render(m[3])
			} else {
				out[i] = sText.Render(l)
			}
		}
	}
	return out
}

func (m *Model) openManual() {
	m.mode = modeManual
	m.manBack = modeForm
	m.manFind = false
	m.manOff = 0
	// Start at the focused option's entry when there is one.
	if r := m.curRow(); r != nil && r.kind == rowOpt {
		name := m.spec.Options[r.opt].Names[0]
		raw := strings.Split(m.spec.Manual, "\n")
		for i, l := range raw {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, name) && (len(t) == len(name) || strings.ContainsAny(t[len(name):len(name)+1], " ,=[")) {
				m.manOff = max(0, i-1)
				break
			}
		}
	}
	m.scrollManual(0)
}

func (m *Model) manHeight() int { return max(1, m.h-4) }

func (m *Model) scrollManual(d int) {
	m.manOff += d
	maxOff := max(0, len(m.manLines)-m.manHeight())
	if m.manOff > maxOff {
		m.manOff = maxOff
	}
	if m.manOff < 0 {
		m.manOff = 0
	}
}

// manSpec is the command whose manual is open.
func (m *Model) manSpec() *manpage.Spec {
	if m.manBack == modeSub {
		return m.toolMsg.spec // opened from the subcommand browser
	}
	return m.spec
}

func (m *Model) findNext(dir int) tea.Cmd {
	q := strings.ToLower(m.manQuery)
	if q == "" {
		return nil
	}
	raw := strings.Split(m.manSpec().Manual, "\n")
	n := len(raw)
	for k := 1; k <= n; k++ {
		i := ((m.manOff+dir*k)%n + n) % n
		if strings.Contains(strings.ToLower(raw[i]), q) {
			m.manOff = i
			m.scrollManual(0)
			return nil
		}
	}
	return m.setStatus("Not found: "+m.manQuery, true)
}

func (m *Model) updateManual(k tea.KeyMsg) tea.Cmd {
	key := k.String()
	if m.manFind {
		switch key {
		case "esc":
			m.manFind = false
			m.manSearch.Blur()
			return nil
		case "enter":
			m.manFind = false
			m.manSearch.Blur()
			m.manQuery = m.manSearch.Value()
			m.manOff--
			return m.findNext(1)
		}
		var cmd tea.Cmd
		m.manSearch, cmd = m.manSearch.Update(k)
		return cmd
	}
	page := m.manHeight()
	switch key {
	case "esc", "q", "ctrl+o", "f1", "?":
		if m.manBack == modeSub {
			return m.openSubs()
		}
		m.mode = modeForm
		return m.focusCurrent()
	case "up", "k", "ctrl+p":
		m.scrollManual(-1)
	case "down", "j", "ctrl+n", "enter":
		m.scrollManual(1)
	case "pgdown", " ", "f", "ctrl+f", "ctrl+d":
		m.scrollManual(page - 1)
	case "pgup", "b", "ctrl+b", "ctrl+u":
		m.scrollManual(-(page - 1))
	case "home", "g":
		m.manOff = 0
	case "end", "G":
		m.scrollManual(len(m.manLines))
	case "/":
		m.manFind = true
		m.manSearch.SetValue("")
		return m.manSearch.Focus()
	case "n":
		return m.findNext(1)
	case "N":
		return m.findNext(-1)
	}
	return nil
}

func (m *Model) viewManual() string {
	w, h := m.w, m.h
	inner := w - 4
	body := make([]string, 0, max(0, h))
	end := min(len(m.manLines), m.manOff+m.manHeight())
	q := m.manQuery
	for _, l := range m.manLines[m.manOff:end] {
		if q != "" {
			l = highlight(l, q)
		}
		body = append(body, ansi.Truncate(l, inner, "…"))
	}
	pct := 100
	if len(m.manLines) > m.manHeight() {
		pct = 100 * m.manOff / max(1, len(m.manLines)-m.manHeight())
	}
	spec := m.manSpec()
	title := sCmd.Render("man "+spec.Command) + sDim.Render(" · ") + sDim.Render(itoa(pct)+"%")
	if spec.Source == "help" {
		title = sCmd.Render(spec.Command+" --help") + sDim.Render(" · "+itoa(pct)+"%")
	}
	out := box(title, body, w, h-1, cViolet)
	var foot string
	if m.manFind {
		foot = " " + m.manSearch.View()
	} else if m.status != "" {
		foot = " " + m.statusView()
	} else {
		foot = keyHelp([][2]string{{"↑↓/jk", "scroll"}, {"space/b", "page"}, {"/", "search"}, {"n/N", "next/prev"}, {"g/G", "top/end"}, {"esc", "back"}})
	}
	return out + "\n" + fit(foot, w)
}

// highlight marks case-insensitive occurrences of q in a styled line.
func highlight(styled, q string) string {
	plain := ansi.Strip(styled)
	lp, lq := strings.ToLower(plain), strings.ToLower(q)
	if !strings.Contains(lp, lq) {
		return styled
	}
	var b strings.Builder
	i := 0
	for {
		j := strings.Index(lp[i:], lq)
		if j < 0 {
			b.WriteString(sText.Render(plain[i:]))
			break
		}
		b.WriteString(sText.Render(plain[i : i+j]))
		b.WriteString(sMatchHL.Render(plain[i+j : i+j+len(q)]))
		i += j + len(q)
	}
	return b.String()
}
