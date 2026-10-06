package manpage

import (
	"compress/gzip"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func names(subs []Subcommand) []string {
	var out []string
	for _, s := range subs {
		out = append(out, s.Name)
	}
	return out
}

func TestHelpSubcommands(t *testing.T) {
	cargo := `Rust's package manager

Usage: cargo [+toolchain] [OPTIONS] [COMMAND]

Options:
  -V, --version                  Print version info and exit
  -h, --help                     Print help

Commands:
    build, b    Compile the current package
    check, c    Analyze the current package
    clean       Remove the target directory
    ...         See all commands with --list

See 'cargo help <command>' for more information on a specific command.
`
	subs := helpSubcommands(cargo)
	if got := names(subs); !reflect.DeepEqual(got, []string{"build", "check", "clean"}) {
		t.Fatalf("cargo = %v", got)
	}
	if !reflect.DeepEqual(subs[0].Aliases, []string{"b"}) || subs[0].Desc != "Compile the current package" || subs[0].Group != "Commands" {
		t.Errorf("build = %+v", subs[0])
	}

	docker := `Usage:  docker [OPTIONS] COMMAND

Common Commands:
  run         Create and run a new container from an image
  ps          List containers

Management Commands:
  buildx*     Docker Buildx
  container   Manage containers

Commands:
  attach      Attach local standard input
  help        Help about any command

Global Options:
      --config string      Location of client config files
`
	subs = helpSubcommands(docker)
	if got := names(subs); !reflect.DeepEqual(got, []string{"run", "ps", "buildx", "container", "attach"}) {
		t.Fatalf("docker = %v", got)
	}
	if subs[2].Group != "Management Commands" {
		t.Errorf("buildx group = %q", subs[2].Group)
	}

	gotool := "Go is a tool for managing Go source code.\n\nUsage:\n\n\tgo <command> [arguments]\n\nThe commands are:\n\n\tbug         start a bug report\n\tbuild       compile packages and dependencies\n\nUse \"go help <command>\" for more information.\n"
	subs = helpSubcommands(gotool)
	if got := names(subs); !reflect.DeepEqual(got, []string{"bug", "build"}) || subs[0].Group != "Commands" {
		t.Fatalf("go = %+v", subs)
	}

	if subs := helpSubcommands("Usage: ls [OPTION]... [FILE]...\n\n  -a, --all    do not ignore entries\n"); subs != nil {
		t.Errorf("ls = %v", subs)
	}
}

func TestManSubcommands(t *testing.T) {
	t.Setenv("COMMANDO_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		path := filepath.Join(dir, name)
		if filepath.Ext(name) == ".gz" {
			f, err := os.Create(path)
			if err != nil {
				t.Fatal(err)
			}
			zw := gzip.NewWriter(f)
			zw.Write([]byte(text))
			zw.Close()
			f.Close()
			return
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("tool.1", ".TH TOOL 1\n.SH NAME\ntool \\- does things\n")
	write("tool-commit.1.gz", ".TH \"TOOL\\-COMMIT\" \"1\"\n.SH \"NAME\"\ntool-commit \\- Record \\fBchanges\\fR to the repository\n.SH \"SYNOPSIS\"\n")
	write("tool-add.1", ".Dd 2024\n.Dt TOOL-ADD 1\n.Sh NAME\n.Nm tool-add\n.Nd add file contents\n")
	write("tool-zap.1", ".TH X 1\n.SH NAME\ntool-zap - remove everything\n")
	write("toolbox.1", ".TH TOOLBOX 1\n.SH NAME\ntoolbox \\- not a subcommand\n")

	manual := `NAME
       tool - does things

HIGH-LEVEL COMMANDS
       Some words about them.

   Main commands
       tool-commit(1)
           Record changes.

       tool-add(1)
           Add files.

SEE ALSO
       tool-commit(1), tool-add(1)
`
	s := &Spec{Command: "tool", Source: "man", Page: filepath.Join(dir, "tool.1"), Manual: manual}
	subs := manSubcommands(s, true)
	want := []Subcommand{
		{Name: "commit", Desc: "Record changes to the repository", Group: "Main commands"},
		{Name: "add", Desc: "add file contents", Group: "Main commands"},
		{Name: "zap", Desc: "remove everything", Group: "Other commands"},
	}
	if !reflect.DeepEqual(subs, want) {
		t.Fatalf("subs =\n%+v\nwant\n%+v", subs, want)
	}
	// The second call is answered from the cache.
	if again := manSubcommands(s, true); !reflect.DeepEqual(again, want) {
		t.Errorf("cached = %+v", again)
	}
	if got := Subcommands(s, true); len(got) != 3 {
		t.Errorf("Subcommands = %v", got)
	}
	// Subcommands of a subcommand aren't listed.
	if got := Subcommands(&Spec{Command: "tool commit", Source: "man", Page: s.Page}, true); got != nil {
		t.Errorf("nested = %v", got)
	}
}

func TestManRefsHeading(t *testing.T) {
	refs := manRefs("GIT COMMANDS\n       git-add(1)\n           Add.\n", "git")
	if len(refs) != 1 || refs[0].Group != "Git commands" {
		t.Errorf("refs = %+v", refs)
	}
}
