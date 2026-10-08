package manpage

import (
	"os"
	"reflect"
	"sort"
	"testing"
)

// Real manuals and --help output in layouts that were once misread. Each
// case lists options that must be read with exactly these names, and the
// least number of options and names the page should give, so a change
// that loses some fails here. The counts are what commando read when the
// file was added; raise them as the parser improves.
func TestFormats(t *testing.T) {
	cases := []struct {
		file, cmd, source string
		options, names    int
		want              [][]string
	}{
		// Names stacked one per line (DocBook).
		{"pg-dump", "pg_dump", "man", 63, 96, [][]string{
			{"-U", "--username"}, {"-F", "--format"}, {"-f", "--file"}, {"-a", "--data-only"},
		}},
		// "-1 --base, -2 --ours, -3 --theirs": three options in one entry;
		// "-X[<param1,param2,...>], --dirstat[=<param1,param2,...>]".
		{"git-diff", "git diff", "man", 83, 105, [][]string{
			{"-1", "--base"}, {"-2", "--ours"}, {"-3", "--theirs"}, {"-X", "--dirstat"},
		}},
		// "-b addr or --bind-address addr" (JDK).
		{"jwebserver", "jwebserver", "man", 6, 12, [][]string{
			{"-b", "--bind-address"}, {"-d", "--directory"}, {"-p", "--port"},
		}},
		// "-T,  --timeout=SECONDS", and long-only options lined up under it.
		{"wget-help", "wget", "help", 153, 197, [][]string{
			{"-T", "--timeout"}, {"--read-timeout"}, {"--no-proxy"}, {"-w", "--wait"},
		}},
		// "-u, --euid <ID,...>", which also hid the option above it.
		{"pgrep-help", "pgrep", "help", 29, 54, [][]string{
			{"-f", "--full"}, {"-u", "--euid"}, {"-g", "--pgroup"}, {"--cgroup"},
		}},
		// "-a --all", "-D --directory=PATH": no comma.
		{"journalctl-help", "journalctl", "help", 60, 85, [][]string{
			{"-a", "--all"}, {"-D", "--directory"}, {"-b", "--boot"}, {"-f", "--follow"},
		}},
		// Settings as subsections named without dashes, with Default: and
		// Type: (npm).
		{"npm-install", "npm install", "man", 25, 25, [][]string{
			{"--save"}, {"--omit"}, {"--install-strategy"}, {"--workspace"},
		}},
		// "-EB --endian=big": a flag standing for one setting.
		{"objdump-help", "objdump", "help", 51, 77, [][]string{
			{"-EB"}, {"-EL"}, {"-b", "--target"},
		}},
	}
	for _, c := range cases {
		b, err := os.ReadFile("testdata/" + c.file + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		s := Parse(c.cmd, Clean(string(b)), c.source)
		names := 0
		for _, o := range s.Options {
			names += len(o.Names)
		}
		if len(s.Options) < c.options || names < c.names {
			t.Errorf("%s: %d options with %d names, want at least %d and %d", c.file, len(s.Options), names, c.options, c.names)
		}
		for _, w := range c.want {
			var got []string
			for _, o := range s.Options {
				if o.HasName(w[0]) {
					got = o.Names
				}
			}
			g, x := append([]string(nil), got...), append([]string(nil), w...)
			sort.Strings(g)
			sort.Strings(x)
			if !reflect.DeepEqual(g, x) {
				t.Errorf("%s: %s has names %v, want %v", c.file, w[0], got, w)
			}
		}
	}
}

func TestConfigOptions(t *testing.T) {
	b, err := os.ReadFile("testdata/npm-install.txt")
	if err != nil {
		t.Fatal(err)
	}
	s := Parse("npm install", Clean(string(b)), "man")
	if s.Summary != "Install a package" {
		t.Errorf("summary = %q", s.Summary)
	}
	save, omit := find(t, s, "--save"), find(t, s, "--omit")
	if save.Kind != KindFlag || omit.Kind != KindChoice || !reflect.DeepEqual(omit.Choices, []string{"dev", "optional", "peer"}) {
		t.Errorf("--save %v, --omit %v %v", save.Kind, omit.Kind, omit.Choices)
	}
	if omit.Label != "Dependency types to omit from the installation tree on disk" {
		t.Errorf("--omit label = %q", omit.Label)
	}
}

// macOS manuals: options named only in the SYNOPSIS (nice, basename), and
// values in braces (pbcopy).
func TestMacOSPages(t *testing.T) {
	cases := []struct {
		file  string
		want  [][]string // names, arg, label
		kinds []Kind
	}{
		{"macos-nice", [][]string{{"-n", "increment", "Increment"}}, []Kind{KindString}},
		{"macos-basename", [][]string{
			{"-a", "", "Every argument is treated as a string as if basename were invoked with …"},
			{"-s", "suffix", "The suffix is taken as its argument, and all other arguments are treate…"},
		}, []Kind{KindFlag, KindString}},
		{"macos-pbcopy", [][]string{
			{"-pboard", "{general|ruler|find|font}", "Specifies which pasteboard to copy to or paste from"},
			{"-Prefer", "{txt|rtf|ps}", "Tells pbpaste what type of data to look for in the pasteboard first"},
		}, []Kind{KindChoice, KindChoice}},
	}
	for _, c := range cases {
		b, err := os.ReadFile("testdata/" + c.file + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		s := Parse(c.file, Clean(string(b)), "man")
		if len(s.Options) != len(c.want) {
			t.Errorf("%s: %d options, want %d", c.file, len(s.Options), len(c.want))
			continue
		}
		for i, w := range c.want {
			o := s.Options[i]
			if o.Names[0] != w[0] || o.Arg != w[1] || o.Label != w[2] || o.Kind != c.kinds[i] {
				t.Errorf("%s: option %d = %v %q %q %v, want %v %v", c.file, i, o.Names, o.Arg, o.Label, o.Kind, w, c.kinds[i])
			}
		}
	}
}
