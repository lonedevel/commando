package ui

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

func TestArgumentFields(t *testing.T) {
	spec := &manpage.Spec{
		Command: "cp",
		Args:    []manpage.Arg{{Name: "SOURCE", Required: true, Path: true}, {Name: "DEST", Required: true, Path: true}},
		Options: []manpage.Option{{Names: []string{"-v"}, Kind: manpage.KindFlag, Label: "Verbose"}},
	}
	m := New(Config{Line: "cp a.txt"})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(loadedMsg{spec: spec, rest: cmdline.Split("a.txt")})

	if len(m.argInputs) != 2 || m.argInputs[0].Value() != "a.txt" || m.argInputs[1].Value() != "" {
		t.Fatalf("arguments not distributed: %d fields", len(m.argInputs))
	}
	if r := m.curRow(); r.kind != rowOpt {
		t.Error("cursor should start on the first option")
	}
	view := ansi.Strip(m.View())
	for _, want := range []string{"Source *", "Dest *"} {
		if !strings.Contains(view, want) {
			t.Errorf("view lacks %q", want)
		}
	}

	// A required argument left empty warns once, then runs on a second Enter.
	keys(m, "enter")
	if m.result.Accepted || !strings.Contains(m.status, "DEST looks required") {
		t.Fatalf("expected a warning, got status %q", m.status)
	}
	keys(m, "up")
	keys(m, "out dir")
	if got := m.command(); got != "cp a.txt out dir" {
		t.Errorf("command = %q", got)
	}
	keys(m, "enter")
	if !m.result.Accepted {
		t.Error("filled-in form should run")
	}
}

func TestDistributeArgs(t *testing.T) {
	file := manpage.Arg{Name: "FILE", Repeat: true}
	pat := manpage.Arg{Name: "PATTERNS", Required: true}
	src := manpage.Arg{Name: "SRC", Repeat: true}
	dst := manpage.Arg{Name: "DEST"}
	cases := []struct {
		args []manpage.Arg
		line string
		want []string
	}{
		{[]manpage.Arg{pat, file}, `TODO src "my dir"`, []string{"TODO", `src "my dir"`}},
		{[]manpage.Arg{src, dst}, "a b c out/", []string{"a b c", "out/"}},
		{[]manpage.Arg{src, dst}, "a", []string{"", "a"}},
		{[]manpage.Arg{dst}, "x y", []string{"x y"}},
		{[]manpage.Arg{pat, file}, "", []string{"", ""}},
	}
	for _, c := range cases {
		if got := distributeArgs(c.args, cmdline.Split(c.line)); !reflect.DeepEqual(got, c.want) {
			t.Errorf("distributeArgs(%q) = %q, want %q", c.line, got, c.want)
		}
	}
}

func TestSubcommandBrowser(t *testing.T) {
	t.Setenv("COMMANDO_DATA_DIR", t.TempDir())
	st := store.Load()
	st.AddRecent("tool commit", "tool commit -a", time.Now().Add(-time.Hour))
	st.AddRecent("ls", "ls -l", time.Now()) // another tool: not listed

	tool := specFrom(t, "gnu-ls")
	tool.Command, tool.Summary = "tool", "does things"
	subs := []manpage.Subcommand{
		{Name: "add", Desc: "Add file contents to the index", Group: "Main"},
		{Name: "commit", Desc: "Record changes", Group: "Main"},
		{Name: "commit-tree", Desc: "Create a new commit object", Group: "Plumbing"},
		{Name: "log", Desc: "Show commit logs", Group: "Main"},
	}
	m := New(Config{Line: "tool", Store: st})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.Update(loadedMsg{spec: tool, cmd: "tool", subs: subs})
	if m.mode != modeSub {
		t.Fatalf("mode = %v, want the browser", m.mode)
	}
	// Own options, then the recent line (selected), then the commands.
	if len(m.subItems) != 6 || m.subItems[0].kind != subOwn || m.subItems[1].entry.Line != "tool commit -a" || m.subSel != 1 {
		t.Fatalf("items = %+v sel=%d", m.subItems, m.subSel)
	}
	view := m.View()
	for _, want := range []string{"Record changes", "tool commit -a", "Plumbing", "its own options"} {
		if !strings.Contains(ansi.Strip(view), want) {
			t.Errorf("view lacks %q", want)
		}
	}

	// Filtering puts name matches before description matches.
	keys(m, "c", "o", "m")
	if m.subItems[m.subSel].kind != subRecent {
		t.Fatalf("selected %+v", m.subItems[m.subSel])
	}
	var got []string
	for _, it := range m.subItems {
		if it.kind == subCmd {
			got = append(got, subs[it.sub].Name)
		}
	}
	if want := []string{"commit", "commit-tree", "log"}; !reflect.DeepEqual(got, want) {
		t.Errorf("filtered = %v, want %v", got, want)
	}
	keys(m, "down", "enter")
	if m.mode != modeLoading || m.cfg.Line != "tool commit" {
		t.Fatalf("enter: mode=%v line=%q", m.mode, m.cfg.Line)
	}

	// The form opens; Esc goes back to the browser, keeping the filter.
	commit := specFrom(t, "gnu-ls")
	commit.Command = "tool commit"
	m.Update(loadedMsg{spec: commit, cmd: "tool commit"})
	if m.mode != modeForm || m.spec.Command != "tool commit" {
		t.Fatalf("form: mode=%v", m.mode)
	}
	if !m.lib {
		t.Error("the command's recent lines were not offered")
	}
	keys(m, "esc", "esc") // close them, then leave the form
	if m.mode != modeSub || m.subFilter.Value() != "com" {
		t.Fatalf("esc: mode=%v filter=%q", m.mode, m.subFilter.Value())
	}

	// A command without an option list falls back to the tool's form.
	keys(m, "down", "enter")
	m.Update(loadedMsg{spec: tool, cmd: "tool", rest: cmdline.Split("commit")})
	if m.mode != modeForm || !strings.Contains(m.status, "No option list for tool commit") {
		t.Errorf("fallback: mode=%v status=%q", m.mode, m.status)
	}
	keys(m, "esc")

	// Errors keep you in the browser.
	keys(m, "enter")
	m.Update(loadedMsg{err: manpage.ErrNotFound, cmd: "tool"})
	if m.mode != modeSub || !m.statusErr {
		t.Errorf("error: mode=%v status=%q", m.mode, m.status)
	}

	// Esc clears the filter, then the tool's own options open its form.
	keys(m, "esc")
	if m.subFilter.Value() != "" {
		t.Fatal("esc did not clear the filter")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyHome})
	keys(m, "enter")
	if m.mode != modeForm || m.spec.Command != "tool" {
		t.Fatalf("own options: mode=%v", m.mode)
	}

	// The manual opened from the browser returns to it.
	keys(m, "esc")
	keys(m, "ctrl+o")
	if m.mode != modeManual || !strings.Contains(m.View(), "man tool") {
		t.Fatalf("manual: mode=%v", m.mode)
	}
	keys(m, "esc")
	if m.mode != modeSub {
		t.Errorf("manual esc: mode=%v", m.mode)
	}

	// Every size renders without overflowing.
	for _, sz := range [][2]int{{60, 12}, {80, 24}, {100, 30}, {200, 60}} {
		m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		lines := strings.Split(m.View(), "\n")
		if len(lines) > sz[1] {
			t.Errorf("%v: %d lines", sz, len(lines))
		}
		for _, l := range lines {
			if w := ansi.StringWidth(l); w > sz[0] {
				t.Errorf("%v: line width %d", sz, w)
			}
		}
	}
}

func TestNoBrowserWithoutSubcommands(t *testing.T) {
	m := New(Config{Line: "ls"})
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m.Update(loadedMsg{spec: specFrom(t, "gnu-ls"), cmd: "ls"})
	if m.mode != modeForm {
		t.Fatalf("mode = %v", m.mode)
	}
	keys(m, "esc")
	if m.mode != modeForm || m.result.Accepted {
		t.Error("esc should quit, not open a browser")
	}
}

// Tiny terminals must not crash any screen.
func TestTinyTerminals(t *testing.T) {
	for _, sz := range [][2]int{{1, 1}, {20, 3}, {30, 6}, {118, 6}, {50, 9}} {
		m := newForm(t, "curl", "curl", 80, 24)
		m.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		keys(m, "down", "space", "ctrl+o", "esc")
		m.Update(tea.KeyMsg{Type: tea.KeyCtrlL})
		_ = m.View()

		b := New(Config{Line: "tool"})
		b.Update(tea.WindowSizeMsg{Width: sz[0], Height: sz[1]})
		b.Update(loadedMsg{spec: specFrom(t, "gnu-ls"), cmd: "tool", subs: []manpage.Subcommand{{Name: "a"}, {Name: "b"}}})
		keys(b, "down", "a", "ctrl+o", "esc")
		_ = b.View()
	}
}

func TestNestedSubcommandBrowser(t *testing.T) {
	spec := func(cmd string) *manpage.Spec {
		s := specFrom(t, "gnu-ls")
		s.Command = cmd
		return s
	}
	m := New(Config{Line: "dock"})
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	m.Update(loadedMsg{spec: spec("dock"), cmd: "dock", subs: []manpage.Subcommand{{Name: "container", Desc: "Manage containers"}, {Name: "run"}}})
	keys(m, "c", "o", "n", "enter")
	if m.cfg.Line != "dock container" {
		t.Fatalf("line = %q", m.cfg.Line)
	}
	// The group has commands of its own: a second list.
	m.Update(loadedMsg{spec: spec("dock container"), cmd: "dock container", subs: []manpage.Subcommand{{Name: "ls"}, {Name: "rm"}}})
	if m.mode != modeSub || m.subTool() != "dock container" || len(m.subStack) != 1 {
		t.Fatalf("second level: mode=%v tool=%q stack=%d", m.mode, m.subTool(), len(m.subStack))
	}
	if v := ansi.Strip(m.View()); !strings.Contains(v, "dock › container") || !strings.Contains(v, "esc back") {
		t.Errorf("breadcrumb or footer missing:\n%s", v)
	}
	keys(m, "l", "s", "enter")
	if m.cfg.Line != "dock container ls" {
		t.Fatalf("line = %q", m.cfg.Line)
	}
	m.Update(loadedMsg{spec: spec("dock container ls"), cmd: "dock container ls"})
	if m.mode != modeForm {
		t.Fatalf("form: mode=%v", m.mode)
	}
	// Esc: back to the group's list, then clear its filter, then up to the tool's.
	keys(m, "esc")
	if m.mode != modeSub || m.subTool() != "dock container" || m.subFilter.Value() != "ls" {
		t.Fatalf("esc 1: mode=%v tool=%q filter=%q", m.mode, m.subTool(), m.subFilter.Value())
	}
	keys(m, "esc", "esc")
	if m.subTool() != "dock" || m.subFilter.Value() != "con" || len(m.subStack) != 0 {
		t.Fatalf("esc 3: tool=%q filter=%q", m.subTool(), m.subFilter.Value())
	}
	if it := m.subItems[m.subSel]; it.kind != subCmd || m.subs[it.sub].Name != "container" {
		t.Errorf("selection not restored: %+v", it)
	}

	// Opening a group directly starts a fresh list: Esc quits from it.
	d := New(Config{Line: "dock container"})
	d.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	d.Update(loadedMsg{spec: spec("dock container"), cmd: "dock container", subs: []manpage.Subcommand{{Name: "ls"}, {Name: "rm"}}})
	if d.mode != modeSub || len(d.subStack) != 0 {
		t.Fatalf("direct: mode=%v stack=%d", d.mode, len(d.subStack))
	}
}

func TestExamples(t *testing.T) {
	m := newForm(t, "bsd-find", "find", 120, 30) // no store: history off
	if len(m.spec.Examples) == 0 {
		t.Fatal("fixture has no examples")
	}
	if m.lib {
		t.Fatal("examples alone shouldn't open the list")
	}
	if !strings.Contains(ansi.Strip(m.View()), "^X examples") {
		t.Error("footer doesn't mention ^X")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	if !m.lib || m.libItems[m.libSel].kind != libExample {
		t.Fatalf("^X: lib=%v sel=%d", m.lib, m.libSel)
	}
	v := ansi.Strip(m.View())
	for _, want := range []string{"Examples from the manual", "Example from the manual", `find / \! -name "*.c" -print`} {
		if !strings.Contains(v, want) {
			t.Errorf("view lacks %q", want)
		}
	}
	keys(m, "d") // can't delete an example
	if !m.lib || !strings.Contains(m.status, "can't be deleted") {
		t.Errorf("d: lib=%v status=%q", m.lib, m.status)
	}
	keys(m, "enter")
	if m.lib || m.command() != `find / \! -name '*.c' -print` {
		t.Errorf("loaded: lib=%v cmd=%q", m.lib, m.command())
	}

	// A manual without examples says so.
	ls := newForm(t, "gnu-ls", "ls", 120, 30)
	ls.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	if ls.lib || !strings.Contains(ls.status, "no examples") {
		t.Errorf("ls: lib=%v status=%q", ls.lib, ls.status)
	}
}

func TestWarnsWhenReordered(t *testing.T) {
	m := newForm(t, "gnu-ls", "ls -l", 120, 30)
	if m.status != "" {
		t.Errorf("plain line: status %q", m.status)
	}
	m = newForm(t, "gnu-ls", "ls /tmp -l", 120, 30)
	if !strings.Contains(m.status, "changed the order") {
		t.Errorf("reordered line: status %q", m.status)
	}
}

func TestValueDescriptions(t *testing.T) {
	m := newForm(t, "bsd-find", "find", 120, 40)
	keys(m, "/", "t", "y", "p", "e", "enter")
	r := m.curRow()
	if r == nil || r.kind != rowOpt || m.spec.Options[r.opt].Names[0] != "-type" {
		t.Fatalf("not on -type: %+v", r)
	}
	v := ansi.Strip(m.View())
	if !strings.Contains(v, "Values:") || !strings.Contains(v, "d  Directory") {
		t.Errorf("help panel lacks value meanings:\n%s", v)
	}
	keys(m, "space") // open the dropdown
	if !m.dropdown || !strings.Contains(ansi.Strip(m.View()), "Symbolic link") {
		t.Errorf("dropdown lacks meanings:\n%s", ansi.Strip(m.View()))
	}
}

func TestFirstClause(t *testing.T) {
	for in, want := range map[string]string{
		"Symbolic link; this is never true if -L": "Symbolic link",
		"Do not sort. Lists entries as found":     "Do not sort",
		"Directory":                               "Directory",
	} {
		if got := firstClause(in); got != want {
			t.Errorf("firstClause(%q) = %q", in, got)
		}
	}
}

func TestApplyTheme(t *testing.T) {
	defer ApplyTheme(lipgloss.DefaultRenderer(), "auto", nil)
	ApplyTheme(lipgloss.DefaultRenderer(), "contrast", map[string]string{"danger": "#FF0000"})
	if cText.Dark != "#FFFFFF" || cRed.Dark != "#FF0000" || cRed.Light != "#FF0000" {
		t.Errorf("contrast: text %v, danger %v", cText, cRed)
	}
	ApplyTheme(lipgloss.DefaultRenderer(), "auto", map[string]string{"accent": "#123456"})
	if cOption.Dark != "#123456" || cText.Dark != "#E5E7EB" {
		t.Errorf("accent: option %v, text %v", cOption, cText)
	}
}

func TestOptionExamplesUI(t *testing.T) {
	m := newForm(t, "curl", "curl", 120, 30)
	keys(m, "ctrl+f", "r", "e", "t", "r", "y", "enter")
	r := m.curRow()
	if r == nil || r.kind != rowOpt || m.spec.Options[r.opt].Names[0] != "--retry" {
		t.Fatalf("not on --retry")
	}
	v := ansi.Strip(m.View())
	if !strings.Contains(v, "Example") || !strings.Contains(v, "curl --retry 7 https://example.com") {
		t.Errorf("help panel lacks the example:\n%s", v)
	}
	// ^X opens the list at this option's example, scrolled into view.
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	it := m.libItems[m.libSel]
	if !m.lib || it.kind != libExample || it.entry.Line != "curl --retry 7 https://example.com" {
		t.Fatalf("^X selected %+v", it)
	}
	if !strings.Contains(ansi.Strip(m.View()), "❯ curl --retry 7") {
		t.Errorf("selection not in view:\n%s", ansi.Strip(m.View()))
	}
	keys(m, "enter")
	if m.command() != "curl --retry 7 https://example.com" {
		t.Errorf("loaded %q", m.command())
	}
}
