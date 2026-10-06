package ui

import (
	"context"
	"reflect"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"

	"github.com/lonedevel/commando/internal/cmdline"
)

func TestSplitLine(t *testing.T) {
	got := splitLine(`find . -name '*|*' | xargs -0 rm 2>&1 && echo "a;b" ; ls &> out & date || true`)
	want := []segment{
		{text: `find . -name '*|*'`},
		{text: `xargs -0 rm 2>&1`, sep: "|"},
		{text: `echo "a;b"`, sep: "&&"},
		{text: `ls &> out`, sep: ";"},
		{text: `date`, sep: "&"},
		{text: `true`, sep: "||"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("splitLine =\n%+v", got)
	}
}

func TestTakeRedirects(t *testing.T) {
	words, notes := takeRedirects(cmdline.Split(`ls -l > out.txt 2>/dev/null >>log < in 2>&1 '>'`))
	var kept []string
	for _, w := range words {
		kept = append(kept, w.Raw)
	}
	if !reflect.DeepEqual(kept, []string{"ls", "-l", "'>'"}) {
		t.Errorf("kept = %v", kept)
	}
	text := ansi.Strip(strings.Join(notes, "\n"))
	for _, want := range []string{"writes output to out.txt, replacing", "discards errors", "appends output to log", "reads input from in", "sends errors where output goes"} {
		if !strings.Contains(text, want) {
			t.Errorf("notes lack %q:\n%s", want, text)
		}
	}
}

func TestExplainParts(t *testing.T) {
	spec := specFrom(t, "gnu-ls")
	parts := cmdline.Explain(spec, cmdline.Split(`-la --color=sometimes --frobnicate /tmp`))
	out := ansi.Strip(strings.Join(explainParts(spec, parts, 100, false), "\n"))
	for _, want := range []string{
		"-l ", "use a long listing format",
		"-a ", "(--all)",
		"--color=sometimes", "not one of the listed values: auto, never, always",
		"--frobnicate", "not in the manual for ls",
		"/tmp", "(argument)",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("explanation lacks %q:\n%s", want, out)
		}
	}
	// Compact: every line fits.
	for _, l := range explainParts(spec, parts, 30, true) {
		if w := ansi.StringWidth(l); w > 30 {
			t.Errorf("compact line too wide (%d): %q", w, ansi.Strip(l))
		}
	}
}

func TestExplainUnknownCommand(t *testing.T) {
	out := ansi.Strip(Explain(context.Background(), "no-such-command-xyz --flag | also-missing-abc", 80, false))
	if !strings.Contains(out, "no-such-command-xyz — no manual") || !strings.Contains(out, "| sends its output into") {
		t.Errorf("Explain =\n%s", out)
	}
}

func TestLibraryExplains(t *testing.T) {
	m := newForm(t, "bsd-find", "find", 120, 40)
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlX})
	v := ansi.Strip(m.View())
	if !strings.Contains(v, "What it does") || !strings.Contains(v, "(argument)") {
		t.Errorf("preview has no explanation:\n%s", v)
	}
}
