package cmdline

import (
	"testing"

	"github.com/lonedevel/commando/internal/manpage"
)

func testSpec() *manpage.Spec {
	return &manpage.Spec{
		Command: "ls",
		Options: []manpage.Option{
			{Names: []string{"-a", "--all"}, Kind: manpage.KindFlag},                                                 // 0
			{Names: []string{"-l"}, Kind: manpage.KindFlag},                                                          // 1
			{Names: []string{"--color"}, Arg: "WHEN", ArgOptional: true, LongEquals: true, Kind: manpage.KindChoice}, // 2
			{Names: []string{"-w", "--width"}, Arg: "COLS", LongEquals: true, Kind: manpage.KindNumber},              // 3
			{Names: []string{"-D"}, Arg: "format", Kind: manpage.KindString},                                         // 4
			{Names: []string{"-v"}, Kind: manpage.KindFlag, Repeatable: true},                                        // 5
			{Names: []string{"--hide"}, Arg: "PATTERN", LongEquals: true, Kind: manpage.KindString},                  // 6
		},
	}
}

func TestQuote(t *testing.T) {
	cases := map[string]string{
		"plain":       "plain",
		"a b":         "'a b'",
		"it's":        `'it'\''s'`,
		"":            "''",
		"~/My Docs":   "~/'My Docs'",
		"*.go":        "'*.go'",
		"--x=1,2":     "--x=1,2",
		"%Y-%m-%d %H": "'%Y-%m-%d %H'",
	}
	for in, want := range cases {
		if got := Quote(in); got != want {
			t.Errorf("Quote(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBuild(t *testing.T) {
	s := testSpec()
	vals := make([]Value, len(s.Options))
	vals[0] = Value{On: true, Count: 1}
	vals[1] = Value{On: true, Count: 1}
	vals[2] = Value{On: true, Text: "auto"}
	vals[3] = Value{On: true, Text: "80"}
	vals[4] = Value{On: true, Text: "%Y %m"}
	vals[5] = Value{On: true, Count: 3}
	vals[6] = Value{On: true, Text: "*.o"}

	got := Render(Build(s, vals, "~/src *.go", false))
	want := `ls -alvvv --color=auto -w 80 -D '%Y %m' --hide='*.o' ~/src *.go`
	if got != want {
		t.Errorf("short:\n got %s\nwant %s", got, want)
	}
	got = Render(Build(s, vals, "", true))
	want = `ls --all -l --color=auto --width=80 -D '%Y %m' -v -v -v --hide='*.o'`
	if got != want {
		t.Errorf("long:\n got %s\nwant %s", got, want)
	}

	// Optional argument left empty: bare flag.
	vals = make([]Value, len(s.Options))
	vals[2] = Value{On: true}
	if got := Render(Build(s, vals, "", false)); got != "ls --color" {
		t.Errorf("optional arg: %s", got)
	}
}

func TestPrefill(t *testing.T) {
	s := testSpec()
	words := Split(`-la --color=never -w 100 -vv -Dfmt --hide '*.o' --unknown ~/src "my file"`)
	vals, args := Prefill(s, words)
	if !vals[0].On || !vals[1].On {
		t.Errorf("-la not applied: %+v", vals)
	}
	if vals[2].Text != "never" || vals[3].Text != "100" || vals[4].Text != "fmt" || vals[6].Text != "*.o" {
		t.Errorf("values: %+v", vals)
	}
	if vals[5].Count != 2 {
		t.Errorf("-vv count = %d", vals[5].Count)
	}
	if args != `--unknown ~/src "my file"` {
		t.Errorf("args = %q", args)
	}
	// Round trip.
	if got := Render(Build(s, vals, args, false)); got != `ls -alvv --color=never -w 100 -D fmt --hide='*.o' --unknown ~/src "my file"` {
		t.Errorf("round trip = %s", got)
	}

	// A cluster with an unknown letter is left alone.
	vals, args = Prefill(s, Split("-lz file"))
	if vals[1].On || args != "-lz file" {
		t.Errorf("unknown cluster: %+v %q", vals, args)
	}
}
