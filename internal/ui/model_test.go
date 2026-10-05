package ui

import (
	"os"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/lonedevel/commando/internal/cmdline"
	"github.com/lonedevel/commando/internal/manpage"
	"github.com/lonedevel/commando/internal/store"
)

func specFrom(t *testing.T, name string) *manpage.Spec {
	t.Helper()
	b, err := os.ReadFile("../manpage/testdata/" + name + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	return manpage.Parse(strings.TrimPrefix(strings.TrimPrefix(name, "gnu-"), "bsd-"), manpage.Clean(string(b)), "man")
}

func newForm(t *testing.T, fixture, line string, w, h int) *Model {
	t.Helper()
	m := New(Config{Line: line})
	m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	words := cmdline.Split(line)
	m.Update(loadedMsg{spec: specFrom(t, fixture), rest: words[1:]})
	if m.mode != modeForm {
		t.Fatalf("mode = %v", m.mode)
	}
	return m
}

func keys(m *Model, ks ...string) {
	for _, k := range ks {
		var msg tea.KeyMsg
		switch k {
		case "up":
			msg = tea.KeyMsg{Type: tea.KeyUp}
		case "down":
			msg = tea.KeyMsg{Type: tea.KeyDown}
		case "left":
			msg = tea.KeyMsg{Type: tea.KeyLeft}
		case "right":
			msg = tea.KeyMsg{Type: tea.KeyRight}
		case "enter":
			msg = tea.KeyMsg{Type: tea.KeyEnter}
		case "esc":
			msg = tea.KeyMsg{Type: tea.KeyEsc}
		case "space":
			msg = tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}}
		case "tab":
			msg = tea.KeyMsg{Type: tea.KeyTab}
		case "ctrl+f":
			msg = tea.KeyMsg{Type: tea.KeyCtrlF}
		case "ctrl+o":
			msg = tea.KeyMsg{Type: tea.KeyCtrlO}
		case "ctrl+s":
			msg = tea.KeyMsg{Type: tea.KeyCtrlS}
		default:
			msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(k)}
		}
		m.Update(msg)
		_ = m.View()
	}
}

func (m *Model) command() string { return cmdline.Render(m.tokens()) }

func TestFormFlow(t *testing.T) {
	m := newForm(t, "gnu-ls", "ls -l /tmp", 120, 40)
	if got := m.command(); got != "ls -l /tmp" {
		t.Fatalf("prefill: %s", got)
	}
	// Starts on the first option (-a); toggle it.
	keys(m, "space")
	if got := m.command(); got != "ls -al /tmp" {
		t.Errorf("toggle: %s", got)
	}
	// Filter to --color, open the dropdown and pick the second value.
	keys(m, "/", "c", "o", "l", "o", "r", "enter")
	if r := m.curRow(); r == nil || r.kind != rowOpt || !m.spec.Options[r.opt].HasName("--color") {
		t.Fatalf("filter did not land on --color")
	}
	keys(m, "space", "down", "down", "enter")
	o := m.spec.Options[m.curRow().opt]
	if got, want := m.command(), "ls -al --color="+o.Choices[1]+" /tmp"; got != want {
		t.Errorf("dropdown: %s, want %s", got, want)
	}
	// Custom value via the last dropdown entry.
	keys(m, "space", "G", "enter", "x", "y", "enter")
	if got := m.command(); got != "ls -al --color=xy /tmp" {
		t.Errorf("custom: %s", got)
	}
	keys(m, "ctrl+s")
	if got := m.command(); got != "ls --all --color=xy -l /tmp" {
		t.Errorf("long names: %s", got)
	}
	keys(m, "esc", "enter") // clear filter, run
	if !m.result.Accepted || !strings.HasPrefix(m.result.Command, "ls --all") {
		t.Errorf("result = %+v", m.result)
	}
}

func TestNumberField(t *testing.T) {
	m := newForm(t, "gnu-ls", "ls", 120, 40)
	keys(m, "/", "w", "i", "d", "t", "h", "enter")
	if r := m.curRow(); !m.spec.Options[r.opt].HasName("--width") {
		t.Fatalf("on %v", m.spec.Options[r.opt].Names)
	}
	keys(m, "8", "q", "+", "+")
	if got := m.command(); got != "ls -w 10" {
		t.Errorf("number: %s (status %q)", got, m.status)
	}
	if !m.statusErr {
		t.Error("letter in number field should warn")
	}
}

func TestRadioGroup(t *testing.T) {
	m := newForm(t, "bsd-ls", "ls -l -x", 120, 40)
	// -x was given last, so it wins over -l.
	if got := m.command(); got != "ls -x" {
		t.Fatalf("radio prefill: %s", got)
	}
	keys(m, "/", "l", "o", "n", "g", " ", "f", "o", "r", "m", "a", "t", "enter")
	r := m.curRow()
	if r.group < 0 {
		t.Fatalf("not on a radio row: %+v %s", r, m.spec.Options[r.opt].Names[0])
	}
	keys(m, "space")
	if got := m.command(); got != "ls "+strings.TrimPrefix(m.spec.Options[r.opt].Names[0], "") {
		t.Errorf("radio select: %s", got)
	}
}

func TestConflicts(t *testing.T) {
	m := newForm(t, "curl", "curl --fail-with-body", 120, 40)
	keys(m, "ctrl+f", "-", "-", "f", "a", "i", "l", "enter")
	r := m.curRow()
	if !m.spec.Options[r.opt].HasName("--fail") {
		t.Fatalf("on %v", m.spec.Options[r.opt].Names)
	}
	keys(m, "space")
	if got := m.command(); got != "curl -f" {
		t.Errorf("conflict: %s", got)
	}
}

func TestViewSizes(t *testing.T) {
	for _, fx := range []string{"gnu-ls", "curl", "gnu-tar", "bsd-find"} {
		for _, sz := range [][2]int{{40, 12}, {60, 20}, {80, 24}, {99, 30}, {100, 30}, {200, 60}} {
			m := newForm(t, fx, "x", sz[0], sz[1])
			for i := 0; i < 30; i++ {
				keys(m, "down")
			}
			keys(m, "space", "down", "up", "ctrl+o", "G", "/", "x", "enter", "esc")
			for _, l := range strings.Split(m.View(), "\n") {
				if w := ansi.StringWidth(l); w > sz[0] {
					t.Fatalf("%s %v: line too wide (%d): %q", fx, sz, w, ansi.Strip(l))
				}
			}
			if n := strings.Count(m.View(), "\n") + 1; n > sz[1] {
				t.Fatalf("%s %v: %d lines", fx, sz, n)
			}
		}
	}
}

func TestPresetsAndRecent(t *testing.T) {
	t.Setenv("COMMANDO_DATA_DIR", t.TempDir())
	st := store.Load()
	st.AddRecent("ls", "ls -l /tmp", time.Now().Add(-time.Hour))

	m := New(Config{Line: "ls", Store: st})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m.Update(loadedMsg{spec: specFrom(t, "gnu-ls")})
	if !m.lib || m.libSel != 1 {
		t.Fatalf("library not offered on open: lib=%v sel=%d", m.lib, m.libSel)
	}
	keys(m, "enter") // load the recent command
	if m.lib || m.command() != "ls -l /tmp" {
		t.Fatalf("recent not applied: lib=%v cmd=%q", m.lib, m.command())
	}

	// Save the current form as a preset.
	keys(m, "space") // toggle -a (cursor starts on the first option)
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlT})
	if !m.naming {
		t.Fatal("^T did not start naming")
	}
	keys(m, "mine", "enter")
	presets, _ := store.Load().For("ls")
	if len(presets) != 1 || presets[0].Name != "mine" || presets[0].Line != "ls -al /tmp" {
		t.Fatalf("saved presets = %+v", presets)
	}

	// Reopen the list, delete the preset.
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
	if !m.lib || m.libItems[m.libSel].kind != libPreset {
		t.Fatalf("^L: lib=%v sel=%d", m.lib, m.libSel)
	}
	keys(m, "d")
	if presets, _ := store.Load().For("ls"); len(presets) != 0 {
		t.Errorf("preset not deleted: %+v", presets)
	}
	keys(m, "esc")
	if m.lib {
		t.Error("esc did not close the list")
	}

	// A command line given up front skips the list.
	m2 := New(Config{Line: "ls -a", Store: store.Load()})
	m2.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m2.Update(loadedMsg{spec: specFrom(t, "gnu-ls"), rest: cmdline.Split("-a")})
	if m2.lib {
		t.Error("library shown despite existing options")
	}
}

func TestFilterPrefersExactCase(t *testing.T) {
	m := newForm(t, "curl", "curl", 120, 40)
	keys(m, "ctrl+f", "-", "X", "enter")
	if r := m.curRow(); !m.spec.Options[r.opt].HasName("-X") {
		t.Errorf("filter -X landed on %v", m.spec.Options[r.opt].Names)
	}
	keys(m, "esc", "ctrl+f", "-", "x", "enter")
	if r := m.curRow(); !m.spec.Options[r.opt].HasName("-x") {
		t.Errorf("filter -x landed on %v", m.spec.Options[r.opt].Names)
	}
}

func TestConfirmRiskyOptions(t *testing.T) {
	spec := &manpage.Spec{Command: "rm", Options: []manpage.Option{
		{Names: []string{"-r"}, Kind: manpage.KindFlag, Label: "Remove recursively", Danger: "deletes directories"},
		{Names: []string{"-v"}, Kind: manpage.KindFlag, Label: "Verbose"},
	}}
	open := func(confirm bool, line string) *Model {
		m := New(Config{Line: line, Confirm: confirm})
		m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
		m.Update(loadedMsg{spec: spec, rest: cmdline.Split(line)[1:]})
		return m
	}

	m := open(true, "rm -r dir")
	keys(m, "enter")
	if m.result.Accepted || len(m.confirming) != 1 || m.confirming[0] != "-r" {
		t.Fatalf("expected a confirmation for -r: %+v %v", m.result, m.confirming)
	}
	if !strings.Contains(ansi.Strip(m.View()), "Run it?") {
		t.Error("confirmation not shown")
	}
	keys(m, "n")
	if m.result.Accepted || m.confirming != nil {
		t.Fatal("declining should return to the form")
	}
	keys(m, "enter", "y")
	if !m.result.Accepted || m.result.Command != "rm -r dir" {
		t.Fatalf("y should run: %+v", m.result)
	}

	// Safe options run straight away, and Confirm=false (print mode) skips it.
	m = open(true, "rm -v dir")
	keys(m, "enter")
	if !m.result.Accepted {
		t.Error("safe command needed confirmation")
	}
	m = open(false, "rm -r dir")
	keys(m, "enter")
	if !m.result.Accepted {
		t.Error("print mode asked for confirmation")
	}
}
