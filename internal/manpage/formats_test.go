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
