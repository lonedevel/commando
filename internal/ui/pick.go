package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/lonedevel/commando/internal/store"
)

const maxSuggestions = 8

// pickItem is a suggestion on the start screen: a command name from
// $PATH, or a saved line (a preset when preset is set).
type pickItem struct {
	text    string
	history bool
	preset  string
}

func (m *Model) updatePick(k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "esc":
		if m.pick.Value() != "" {
			m.pick.SetValue("")
			m.updateSuggestions()
			return nil
		}
		m.result = Result{}
		return tea.Quit
	case "enter":
		line := strings.TrimSpace(m.pick.Value())
		if it := m.pickSel(); it != nil && (it.history || !strings.Contains(line, " ")) {
			line = it.text
		}
		if line == "" {
			return nil
		}
		m.cfg.Line = line
		m.mode = modeLoading
		m.loadWhat = strings.Fields(line)[0]
		m.pickErr = ""
		m.pick.Blur()
		return tea.Batch(m.spin.Tick, m.load(line))
	case "tab":
		it := m.pickSel()
		if it == nil && len(m.sugg) > 0 {
			it = &m.sugg[0]
		}
		if it != nil {
			text := it.text
			if !it.history {
				text += " "
			}
			m.pick.SetValue(text)
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

// pickSel is the chosen suggestion, or nil.
func (m *Model) pickSel() *pickItem {
	if m.suggSel < 0 || m.suggSel >= len(m.sugg) {
		return nil
	}
	return &m.sugg[m.suggSel]
}

// updateSuggestions lists command names starting with what's typed, and
// the saved lines of every command that contain all of its words. With
// nothing typed, it lists the most recent lines.
func (m *Model) updateSuggestions() {
	q := m.pick.Value()
	m.sugg = nil
	m.suggSel = -1
	var history []pickItem
	if m.cfg.Store != nil {
		var found []store.Entry
		if strings.TrimSpace(q) == "" {
			found = m.cfg.Store.AllRecent(maxSuggestions)
		} else {
			found = m.cfg.Store.Search(strings.Fields(q), maxSuggestions)
		}
		for _, e := range found {
			history = append(history, pickItem{text: e.Line, history: true, preset: e.Name})
		}
	}
	if strings.TrimSpace(q) == "" {
		m.sugg = history
		if len(m.sugg) > 0 {
			m.suggSel = 0 // the latest command, ready to open again
		}
		return
	}
	if !strings.Contains(q, " ") {
		var exact, pre []string
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
		names := append(exact, pre...)
		limit := maxSuggestions
		if len(history) > 0 {
			limit = 4 // leave room for the history
		}
		for _, c := range names[:min(len(names), limit)] {
			m.sugg = append(m.sugg, pickItem{text: c})
		}
	}
	m.sugg = append(m.sugg, history...)
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
