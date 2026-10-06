package cmdline

import (
	"reflect"
	"testing"

	"github.com/lonedevel/commando/internal/manpage"
)

func TestExplain(t *testing.T) {
	spec := &manpage.Spec{
		Command:  "tar",
		Synopsis: "tar [OPTION...] [FILE]...",
		Args:     []manpage.Arg{{Name: "FILE", Repeat: true}},
		Options: []manpage.Option{
			{Names: []string{"-c", "--create"}},
			{Names: []string{"-z", "--gzip"}},
			{Names: []string{"-f", "--file"}, Arg: "ARCHIVE"},
			{Names: []string{"--exclude"}, Arg: "PATTERN", LongEquals: true},
		},
	}
	got := Explain(spec, Split(`-czf backup.tgz --exclude='*.o' --bogus src -- -weird`))
	want := []Part{
		{Text: "-c", Kind: PartOption, Opt: 0},
		{Text: "-z", Kind: PartOption, Opt: 1},
		{Text: "-f backup.tgz", Kind: PartOption, Opt: 2, Value: "backup.tgz"},
		{Text: "--exclude='*.o'", Kind: PartOption, Opt: 3, Value: "*.o"},
		{Text: "--bogus", Kind: PartUnknown},
		{Text: "src", Kind: PartArg, Arg: "FILE"},
		{Text: "--", Kind: PartEnd},
		{Text: "-weird", Kind: PartArg, Arg: "FILE"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Explain =\n%+v\nwant\n%+v", got, want)
	}

	// A lone option is kept as typed.
	if got := Explain(spec, Split(`-f"x y"`)); len(got) != 1 || got[0].Text != `-f"x y"` || got[0].Value != "x y" {
		t.Errorf("attached value = %+v", got)
	}
}

func TestExplainExpression(t *testing.T) {
	spec := &manpage.Spec{
		Command:  "find",
		Synopsis: "find [-H] [starting-point...] [expression]",
		Args:     []manpage.Arg{{Name: "starting-point", Repeat: true}, {Name: "expression"}},
		Options: []manpage.Option{
			{Names: []string{"-name"}, Arg: "pattern"},
			{Names: []string{"-type"}, Arg: "c"},
		},
	}
	var kinds []PartKind
	var args []string
	for _, p := range Explain(spec, Split(`. /tmp \( -name a -o -name b \) ! -type d`)) {
		kinds = append(kinds, p.Kind)
		if p.Kind == PartArg {
			args = append(args, p.Arg)
		}
	}
	want := []PartKind{PartArg, PartArg, PartOperator, PartOption, PartOperator, PartOption, PartOperator, PartOperator, PartOption}
	if !reflect.DeepEqual(kinds, want) {
		t.Errorf("kinds = %v", kinds)
	}
	// The paths are starting points; nothing is named "expression".
	if !reflect.DeepEqual(args, []string{"starting-point", "starting-point"}) {
		t.Errorf("args = %v", args)
	}
}

func TestArgNames(t *testing.T) {
	args := []manpage.Arg{{Name: "SOURCE", Repeat: true}, {Name: "DEST"}}
	if got := argNames(args, 3); !reflect.DeepEqual(got, []string{"SOURCE", "SOURCE", "DEST"}) {
		t.Errorf("3 = %v", got)
	}
	if got := argNames([]manpage.Arg{{Name: "A"}}, 2); !reflect.DeepEqual(got, []string{"A", ""}) {
		t.Errorf("extra = %v", got)
	}
}
