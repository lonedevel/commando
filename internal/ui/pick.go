package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const maxSuggestions = 8

func (m *Model) updatePick(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		m.result = Result{}
		return tea.Quit
	case "enter":
		line := strings.TrimSpace(m.pick.Value())
		if line == "" && m.suggRecent && m.suggSel >= 0 && m.suggSel < len(m.sugg) {
			line = m.sugg[m.suggSel]
		}
		if line == "" {
			return nil
		}
		if m.suggSel >= 0 && m.suggSel < len(m.sugg) && !strings.Contains(line, " ") {
			line = m.sugg[m.suggSel]
		}
		m.cfg.Line = line
		m.mode = modeLoading
		m.loadWhat = strings.Fields(line)[0]
		m.pickErr = ""
		m.pick.Blur()
		return tea.Batch(m.spin.Tick, m.load(line))
	case "tab":
		if len(m.sugg) > 0 {
			m.pick.SetValue(m.sugg[max(0, m.suggSel)] + " ")
			m.pick.CursorEnd()
			m.updateSuggestions()
		}
		return nil
	case "down", "ctrl+n":
		if len(m.sugg) > 0 {
			m.suggSel = (m.suggSel + 1) % len(m.sugg)
		}
		return nil
	case "up", "ctrl+p":
		if len(m.sugg) > 0 {
			if m.suggSel <= 0 {
				m.suggSel = len(m.sugg)
			}
			m.suggSel--
		}
		return nil
	}
	var cmd tea.Cmd
	m.pick, cmd = m.pick.Update(k)
	m.updateSuggestions()
	return cmd
}

func (m *Model) updateSuggestions() {
	q := m.pick.Value()
	m.sugg = nil
	m.suggSel = -1
	m.suggRecent = false
	if q == "" && m.cfg.Store != nil {
		// Nothing typed yet: offer recently run commands.
		for _, e := range m.cfg.Store.AllRecent(maxSuggestions) {
			m.sugg = append(m.sugg, e.Line)
		}
		if len(m.sugg) > 0 {
			m.suggRecent = true
			m.suggSel = 0
		}
		return
	}
	if q == "" || strings.Contains(q, " ") {
		return
	}
	var exact []string
	var pre []string
	for _, c := range m.pathCmds {
		if c == q {
			exact = append(exact, c)
		} else if strings.HasPrefix(c, q) {
			pre = append(pre, c)
		}
	}
	sort.Slice(pre, func(i, j int) bool {
		if len(pre[i]) != len(pre[j]) {
			return len(pre[i]) < len(pre[j])
		}
		return pre[i] < pre[j]
	})
	m.sugg = append(exact, pre...)
	if len(m.sugg) > maxSuggestions {
		m.sugg = m.sugg[:maxSuggestions]
	}
}

// complete performs filesystem completion on the input's last word.
// It returns done=false when there was nothing to complete, so Tab can
// fall back to moving focus.
func (m *Model) complete(ti *textinput.Model, lastWord bool) (bool, tea.Cmd) {
	val := ti.Value()
	prefix, word := "", val
	if lastWord {
		if i := strings.LastIndexAny(val, " \t"); i >= 0 {
			prefix, word = val[:i+1], val[i+1:]
		}
	}
	if word == "" && !lastWord {
		word = "./"
	}
	if word == "" {
		return false, nil
	}
	expanded := word
	if strings.HasPrefix(word, "~/") {
		if home, err := os.UserHomeDir(); err == nil {
			expanded = filepath.Join(home, word[2:])
			if strings.HasSuffix(word, "/") {
				expanded += "/"
			}
		}
	}
	dir, base := filepath.Split(expanded)
	readDir := dir
	if readDir == "" {
		readDir = "."
	}
	ents, err := os.ReadDir(readDir)
	if err != nil {
		return false, nil
	}
	var matches []string
	for _, e := range ents {
		n := e.Name()
		if !strings.HasPrefix(n, base) || (strings.HasPrefix(n, ".") && !strings.HasPrefix(base, ".")) {
			continue
		}
		if e.IsDir() {
			n += "/"
		} else if e.Type()&os.ModeSymlink != 0 {
			if st, err := os.Stat(filepath.Join(readDir, n)); err == nil && st.IsDir() {
				n += "/"
			}
		}
		matches = append(matches, n)
	}
	if len(matches) == 0 {
		return true, m.setStatus("No matches for "+word, true)
	}
	sort.Strings(matches)
	common := matches[0]
	for _, x := range matches[1:] {
		for !strings.HasPrefix(x, common) {
			common = common[:len(common)-1]
		}
	}
	// Rebuild using the user's spelling of the directory part.
	wdir, _ := filepath.Split(word)
	if word == "./" && dir == "./" {
		wdir = ""
	}
	ti.SetValue(prefix + wdir + common)
	ti.CursorEnd()
	if r := m.curRow(); r != nil && r.kind == rowOpt {
		m.syncText(r.opt)
	}
	if len(matches) > 1 {
		show := matches
		if len(show) > 12 {
			show = append(show[:12:12], "…")
		}
		return true, m.setStatus(strings.Join(show, "  "), false)
	}
	return true, nil
}
