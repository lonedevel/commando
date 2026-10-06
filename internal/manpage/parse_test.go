package manpage

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/lonedevel/commando/internal/complete"
)

func load(t *testing.T, name string) *Spec {
	t.Helper()
	b, err := os.ReadFile("testdata/" + name + ".txt")
	if err != nil {
		t.Fatal(err)
	}
	return Parse(name, Clean(string(b)), "man")
}

func find(t *testing.T, s *Spec, name string) *Option {
	t.Helper()
	for i := range s.Options {
		if s.Options[i].HasName(name) {
			return &s.Options[i]
		}
	}
	t.Fatalf("%s: option %s not found", s.Command, name)
	return nil
}

func groupNames(s *Spec, g Group) []string {
	var out []string
	for _, m := range g.Members {
		out = append(out, s.Options[m].Names[0])
	}
	return out
}

func TestClean(t *testing.T) {
	in := "N\bNA\bAM\bME\bE\n  _\bf_\bi_\bl_\be\x1b[1m!\x1b[0m\tx\n"
	want := "NAME\n  file! x\n"
	if got := Clean(in); got != want {
		t.Errorf("Clean = %q, want %q", got, want)
	}
}

func TestGNULs(t *testing.T) {
	s := load(t, "gnu-ls")
	if s.Summary != "list directory contents" {
		t.Errorf("summary = %q", s.Summary)
	}
	if s.Synopsis != "ls [OPTION]... [FILE]..." {
		t.Errorf("synopsis = %q", s.Synopsis)
	}
	if !strings.HasPrefix(s.Description, "List information about the FILEs") {
		t.Errorf("description = %q", s.Description)
	}
	if len(s.Options) < 55 {
		t.Errorf("got %d options, want >= 55", len(s.Options))
	}

	a := find(t, s, "--all")
	if !reflect.DeepEqual(a.Names, []string{"-a", "--all"}) || a.Kind != KindFlag {
		t.Errorf("--all = %+v", a)
	}
	if a.Label != "Do not ignore entries starting with ." {
		t.Errorf("--all label = %q", a.Label)
	}

	c := find(t, s, "--color")
	if c.Kind != KindChoice || !c.ArgOptional || c.Arg != "WHEN" {
		t.Errorf("--color = %+v", c)
	}
	for _, v := range []string{"always", "auto", "never"} {
		if indexOfStr(c.Choices, v) < 0 {
			t.Errorf("--color choices %v missing %s", c.Choices, v)
		}
	}

	w := find(t, s, "--width")
	if w.Kind != KindNumber || !w.LongEquals || w.Arg != "COLS" {
		t.Errorf("--width = %+v", w)
	}

	sort := find(t, s, "--sort")
	want := []string{"none", "size", "time", "version", "extension", "width"}
	if !reflect.DeepEqual(sort.Choices, want) {
		t.Errorf("--sort choices = %v, want %v", sort.Choices, want)
	}
	if f := find(t, s, "--format"); f.Label != "Format" || len(f.Choices) != 7 {
		t.Errorf("--format = %q %v", f.Label, f.Choices)
	}
	// Tags that exactly fill the indent: "--zero end each output line".
	if z := find(t, s, "--zero"); z.Kind != KindFlag || !strings.HasPrefix(z.Desc, "end each output line") {
		t.Errorf("--zero = %+v", z)
	}
	if h := find(t, s, "--hide"); h.Kind != KindString {
		t.Errorf("--hide kind = %v", h.Kind)
	}
}

func TestBSDLs(t *testing.T) {
	s := load(t, "bsd-ls")
	if s.Summary != "list directory contents" {
		t.Errorf("summary = %q", s.Summary)
	}
	if len(s.Options) < 40 {
		t.Errorf("got %d options", len(s.Options))
	}
	for _, n := range []string{"-@", "-%", "-,", "-1"} {
		find(t, s, n)
	}
	d := find(t, s, "-D")
	if d.Arg != "format" || d.Kind != KindString {
		t.Errorf("-D = %+v", d)
	}
	c := find(t, s, "--color")
	if !reflect.DeepEqual(c.Choices, []string{"always", "auto", "never"}) {
		t.Errorf("--color choices = %v", c.Choices)
	}
	l := find(t, s, "-l")
	if !strings.HasPrefix(l.Label, "List files in the long format") {
		t.Errorf("-l label = %q", l.Label)
	}

	groups := map[string][]string{}
	for _, g := range s.Groups {
		groups[g.Label] = groupNames(s, g)
	}
	if got := groups["Format"]; !reflect.DeepEqual(got, []string{"-C", "-l", "-x", "-1"}) {
		t.Errorf("Format group = %v (all: %v)", got, groups)
	}
	if got := groups["Sort order"]; !reflect.DeepEqual(got, []string{"-S", "-t"}) {
		t.Errorf("Sort order group = %v", got)
	}
	if got := find(t, s, "-L").Conflicts; indexOfStr(got, "-P") < 0 {
		t.Errorf("-L conflicts = %v, want -P", got)
	}
}

func TestGNUGrep(t *testing.T) {
	s := load(t, "gnu-grep")
	if !strings.Contains(s.Synopsis, "\n") {
		t.Errorf("want multi-line synopsis, got %q", s.Synopsis)
	}
	if o := find(t, s, "--regexp"); o.Section != "Matching Control" || o.Arg != "PATTERNS" {
		t.Errorf("--regexp = %+v", o)
	}
	if o := find(t, s, "--file"); o.Kind != KindPath {
		t.Errorf("--file kind = %v", o.Kind)
	}
	if o := find(t, s, "--max-count"); o.Kind != KindNumber {
		t.Errorf("--max-count kind = %v", o.Kind)
	}
	if o := find(t, s, "--binary-files"); !reflect.DeepEqual(o.Choices, []string{"binary", "without-match", "text"}) {
		t.Errorf("--binary-files choices = %v", o.Choices)
	}
	if o := find(t, s, "--directories"); !reflect.DeepEqual(o.Choices, []string{"read", "skip", "recurse"}) {
		t.Errorf("--directories choices = %v", o.Choices)
	}
	if o := find(t, s, "--colour"); o.Kind != KindChoice {
		t.Errorf("--colour = %+v", o)
	}
	// "-NUM" pseudo option is dropped from -C NUM, -NUM, --context=NUM.
	if o := find(t, s, "--context"); !reflect.DeepEqual(o.Names, []string{"-C", "--context"}) {
		t.Errorf("--context names = %v", o.Names)
	}
}

func TestGNUTar(t *testing.T) {
	s := load(t, "gnu-tar")
	var mode *Group
	for i := range s.Groups {
		if s.Groups[i].Label == "Operation mode" {
			mode = &s.Groups[i]
		}
	}
	if mode == nil {
		t.Fatalf("no Operation mode group: %+v", s.Groups)
	}
	names := groupNames(s, *mode)
	for _, n := range []string{"-c", "-x", "-t"} {
		if indexOfStr(names, n) < 0 {
			t.Errorf("operation mode %v missing %s", names, n)
		}
	}
	if o := find(t, s, "--file"); o.Kind != KindPath {
		t.Errorf("--file = %+v", o)
	}
}

func TestCurl(t *testing.T) {
	s := load(t, "curl")
	if len(s.Options) < 240 {
		t.Errorf("got %d options", len(s.Options))
	}
	d := find(t, s, "--data")
	if d.Arg != "data" || !reflect.DeepEqual(d.Names, []string{"-d", "--data"}) {
		t.Errorf("--data = %+v", d)
	}
	find(t, s, "-#")
	if o := find(t, s, "--max-time"); o.Kind != KindNumber {
		t.Errorf("--max-time kind = %v", o.Kind)
	}
	if o := find(t, s, "--output"); o.Kind != KindPath {
		t.Errorf("--output kind = %v", o.Kind)
	}
	// Sample values ("200K, 3m and 1G") are not choices.
	if o := find(t, s, "--limit-rate"); o.Kind == KindChoice {
		t.Errorf("--limit-rate choices = %v", o.Choices)
	}
	if o := find(t, s, "--fail"); indexOfStr(o.Conflicts, "--fail-with-body") < 0 {
		t.Errorf("--fail conflicts = %v", o.Conflicts)
	}
	// Body text that starts with an option ("--no-option. That is, ...")
	// must not become an option.
	for _, o := range s.Options {
		if strings.HasSuffix(o.Names[0], ".") {
			t.Errorf("bogus option %v", o.Names)
		}
	}
}

func TestBSDFindAndSed(t *testing.T) {
	f := load(t, "bsd-find")
	if o := find(t, f, "-Bnewer"); o.Arg != "file" || !strings.HasPrefix(o.Desc, "Same as -newerBm") {
		t.Errorf("-Bnewer = %+v", o)
	}
	if o := find(t, f, "-maxdepth"); o.Kind != KindNumber && o.Kind != KindString {
		t.Errorf("-maxdepth = %+v", o)
	}
	s := load(t, "bsd-sed")
	if o := find(t, s, "-i"); o.Arg != "extension" {
		t.Errorf("-i = %+v", o)
	}
	if o := find(t, s, "-f"); o.Kind != KindPath {
		t.Errorf("-f = %+v", o)
	}
}

func TestHelpOutput(t *testing.T) {
	help := `Usage: cargo [+toolchain] [OPTIONS] [COMMAND]

Options:
  -V, --version                  Print version info and exit
      --color <WHEN>             Coloring [possible values: auto, always, never]
  -C <DIRECTORY>                 Change to DIRECTORY before doing anything
  -v, --verbose...               Use verbose output (-vv very verbose/build.rs output)
      --edition <E>              Edition
      --crate-type <bin|lib|rlib>  Crate type

Commands:
    build, b    Compile the current package
`
	s := Parse("cargo", Clean(help), "help")
	if s.Synopsis != "Usage: cargo [+toolchain] [OPTIONS] [COMMAND]" {
		t.Errorf("synopsis = %q", s.Synopsis)
	}
	c := find(t, s, "--color")
	if c.Label != "Coloring" || !reflect.DeepEqual(c.Choices, []string{"auto", "always", "never"}) {
		t.Errorf("--color = %+v", c)
	}
	if o := find(t, s, "-C"); o.Kind != KindPath || o.Section != "Options" {
		t.Errorf("-C = %+v", o)
	}
	if o := find(t, s, "--verbose"); !o.Repeatable || o.Kind != KindFlag {
		t.Errorf("--verbose = %+v", o)
	}
	if o := find(t, s, "--crate-type"); !reflect.DeepEqual(o.Choices, []string{"bin", "lib", "rlib"}) {
		t.Errorf("--crate-type = %+v", o)
	}
	if !listsCommand(help, "build") || listsCommand(help, "colour") {
		t.Error("listsCommand")
	}
}

func TestParseTag(t *testing.T) {
	cases := []struct {
		tag      string
		names    []string
		arg      string
		optional bool
		ok       bool
	}{
		{"-a, --all", []string{"-a", "--all"}, "", false, true},
		{"--color[=WHEN]", []string{"--color"}, "WHEN", true, true},
		{"-w, --width=COLS", []string{"-w", "--width"}, "COLS", false, true},
		{"-D format", []string{"-D"}, "format", false, true},
		{"-d, --data <data>", []string{"-d", "--data"}, "data", false, true},
		{"-i[SUFFIX], --in-place[=SUFFIX]", []string{"-i", "--in-place"}, "SUFFIX", true, true},
		{"-o key=val[,key=val]", []string{"-o"}, "key=val[,key=val]", false, true},
		{"--no-option. That is", nil, "", false, false},
		{"-l option is given", nil, "", false, false},
		{"--", nil, "", false, false},
	}
	for _, c := range cases {
		ti, ok := parseTag(c.tag)
		if ok != c.ok {
			t.Errorf("parseTag(%q) ok = %v", c.tag, ok)
			continue
		}
		if !ok {
			continue
		}
		if !reflect.DeepEqual(ti.names, c.names) || ti.arg != c.arg || ti.optional != c.optional {
			t.Errorf("parseTag(%q) = %+v", c.tag, ti)
		}
	}
}

func indexOfStr(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

func TestApplyCompletions(t *testing.T) {
	s := load(t, "gnu-ls")
	res := complete.Result{
		Values: complete.Values{
			"--block-size": {"K", "M", "G"},
			"--sort":       {"size", "time", "status"},
			"-w":           {"1", "2"}, // number options stay numbers
		},
		Sources: map[string]string{"--block-size": "zsh", "--sort": "zsh and fish", "-w": "zsh"},
	}
	ApplyCompletions(s, res)
	if o := find(t, s, "--block-size"); o.Kind != KindChoice || o.ChoiceSource != "zsh" || !reflect.DeepEqual(o.Choices, []string{"K", "M", "G"}) {
		t.Errorf("--block-size = %+v", o)
	}
	// Values from the manual are kept after the completion's, and a
	// completion value the manual never mentions ("status") is left out.
	want := []string{"size", "time", "none", "version", "extension", "width"}
	if o := find(t, s, "--sort"); !reflect.DeepEqual(o.Choices, want) || o.ChoiceSource != "zsh and fish" {
		t.Errorf("--sort = %v (%s)", o.Choices, o.ChoiceSource)
	}
	if o := find(t, s, "--width"); o.Kind != KindNumber || o.ChoiceSource != "" {
		t.Errorf("--width = %+v", o)
	}
}

func TestDanger(t *testing.T) {
	cases := map[string]string{
		"remove directories and their contents recursively":         "can delete or overwrite data",
		"Recursively remove all files in the directory.":            "can delete or overwrite data",
		"receiver deletes before xfer, not during":                  "can delete or overwrite data",
		"Attempt to remove the file hierarchy rooted in each file.": "can delete or overwrite data",
		"ignore nonexistent files and arguments, never prompt":      "skips the confirmation prompt",
		"force deletion of dirs even if not empty":                  "forces deleting or overwriting",
		"prompt before every removal":                               "",
		"do not overwrite an existing file":                         "",
		"Never remove the root directory.":                          "",
		"remove any trailing slashes from each SOURCE argument":     "",
		"Overwrite metadata of existing directories (default).":     "",
		"(TLS) Forces curl to use TLS version 1.2 or later.":        "",
		"Display the number of blocks used. Files are removed.":     "",
		// Cleaning up after itself (docker run --rm, docker build --rm).
		"Automatically remove the container and its associated anonymous volumes when it exits": "",
		"Remove intermediate containers after a successful build":                               "",
	}
	for desc, want := range cases {
		if got := dangerFromText(desc); got != want {
			t.Errorf("dangerFromText(%q) = %q, want %q", desc, got, want)
		}
	}

	s := load(t, "gnu-tar")
	ApplyDanger(s)
	for name, risky := range map[string]bool{"--remove-files": true, "--overwrite": true, "--unlink-first": true, "--overwrite-dir": false, "--create": false, "--extract": false} {
		if got := find(t, s, name).Danger != ""; got != risky {
			t.Errorf("tar %s risky = %v, want %v", name, got, risky)
		}
	}

	// Known options are flagged even when the wording isn't destructive,
	// and commands like shred only use the table.
	rm := &Spec{Command: "rm", Options: []Option{{Names: []string{"-f", "--force"}, Desc: "ignore nonexistent files"}}}
	ApplyDanger(rm)
	if rm.Options[0].Danger != "deletes without asking, even read-only files" {
		t.Errorf("rm -f danger = %q", rm.Options[0].Danger)
	}
	shred := &Spec{Command: "shred", Options: []Option{
		{Names: []string{"-n"}, Desc: "overwrite N times instead of the default"},
		{Names: []string{"-u"}, Desc: "deallocate and remove file after overwriting"},
	}}
	ApplyDanger(shred)
	if shred.Options[0].Danger != "" || shred.Options[1].Danger == "" {
		t.Errorf("shred: -n %q, -u %q", shred.Options[0].Danger, shred.Options[1].Danger)
	}
}

func TestParseArgs(t *testing.T) {
	type A = Arg
	cases := []struct {
		synopsis, command string
		want              []Arg
	}{
		{"cp [OPTION]... [-T] SOURCE DEST", "cp", []A{{Name: "SOURCE", Required: true, Path: true}, {Name: "DEST", Required: true, Path: true}}},
		{"rm [OPTION]... [FILE]...", "rm", []A{{Name: "FILE", Repeat: true, Path: true}}},
		{"grep [OPTION...] PATTERNS [FILE...]\ngrep [OPTION...] -e PATTERNS ... [FILE...]", "grep", []A{{Name: "PATTERNS", Required: true}, {Name: "FILE", Repeat: true, Path: true}}},
		{"chmod [OPTION]... MODE[,MODE]... FILE...", "chmod", []A{{Name: "MODE", Required: true, Repeat: true}, {Name: "FILE", Required: true, Repeat: true, Path: true}}},
		{"ls [-@ABC1%,] [--color=when] [-D format] [file ...]", "ls", []A{{Name: "file", Repeat: true, Path: true}}},
		{"cp [-R [-H | -L | -P]] [-f | -i | -n] [-alNpsvx] source_file target_file", "cp", []A{{Name: "source_file", Required: true, Path: true}, {Name: "target_file", Required: true, Path: true}}},
		{"find [-H | -L | -P] [-EXdsx] [-f path] path ... [expression]", "find", []A{{Name: "path", Required: true, Repeat: true, Path: true}, {Name: "expression"}}},
		{"git commit [-a | --interactive] [--] [<pathspec>...]", "git commit", []A{{Name: "pathspec", Repeat: true, Path: true}}},
		{"Usage: rustc [OPTIONS] INPUT", "rustc", []A{{Name: "INPUT", Required: true}}},
		// Too irregular: keep the single Arguments field.
		{"tar {A|c|d|r|t|u|x}[GnSkUWOmpsMBiajJzZhPlRvwo] [ARG...]", "tar", nil},
		{"curl [options / URLs]", "curl", nil},
		{"Local: rsync [OPTION...] SRC... [DEST]", "rsync", nil},
		{"ssh [-46] destination [command [argument ...]]", "ssh", nil},
		{"other [OPTION]... FILE", "ls", nil},
	}
	for _, c := range cases {
		if got := parseArgs(c.synopsis, c.command); !reflect.DeepEqual(got, c.want) {
			t.Errorf("parseArgs(%q) =\n %+v\nwant\n %+v", c.synopsis, got, c.want)
		}
	}
	if l := (Arg{Name: "LINK_NAME"}).Label(); l != "Link name" {
		t.Errorf("Label = %q", l)
	}

	// From whole pages: BSD ls by its real name.
	b, _ := os.ReadFile("testdata/bsd-ls.txt")
	s := Parse("ls", Clean(string(b)), "man")
	if len(s.Args) != 1 || s.Args[0].Name != "file" || s.Args[0].Required || !s.Args[0].Repeat {
		t.Errorf("BSD ls args = %+v", s.Args)
	}
}
