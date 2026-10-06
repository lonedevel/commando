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
	subs := manSubcommands(t.Context(), s, true)
	want := []Subcommand{
		{Name: "commit", Desc: "Record changes to the repository", Group: "Main commands"},
		{Name: "add", Desc: "add file contents", Group: "Main commands"},
		{Name: "zap", Desc: "remove everything", Group: "Other commands"},
	}
	if !reflect.DeepEqual(subs, want) {
		t.Fatalf("subs =\n%+v\nwant\n%+v", subs, want)
	}
	// The second call is answered from the cache.
	if again := manSubcommands(t.Context(), s, true); !reflect.DeepEqual(again, want) {
		t.Errorf("cached = %+v", again)
	}
	if got := Subcommands(t.Context(), s, true); len(got) != 3 {
		t.Errorf("Subcommands = %v", got)
	}
	// Subcommands of a subcommand aren't listed.
	if got := Subcommands(t.Context(), &Spec{Command: "tool commit", Source: "man", Page: s.Page}, true); got != nil {
		t.Errorf("nested = %v", got)
	}
}

func TestManRefsHeading(t *testing.T) {
	refs := manRefs("GIT COMMANDS\n       git-add(1)\n           Add.\n", "git")
	if len(refs) != 1 || refs[0].Group != "Git commands" {
		t.Errorf("refs = %+v", refs)
	}
}

func TestNestedManSubcommands(t *testing.T) {
	t.Setenv("COMMANDO_CACHE_DIR", t.TempDir())
	dir := t.TempDir()
	write := func(name, text string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	page := func(name, desc string) string { return ".TH X 1\n.SH NAME\n" + name + " \\- " + desc + "\n" }
	write("dock.1", page("dock", "runs things"))
	write("dock-run.1", page("dock-run", "run a container"))
	// container groups ls and rm, and its page says so.
	write("dock-container.1", page("dock-container", "manage containers")+".SH SEE ALSO\n\\fBdock\\-container\\-ls(1)\\fR, \\fBdock\\-container\\-rm(1)\\fR\n")
	write("dock-container-ls.1", page("dock-container-ls", "list containers"))
	write("dock-container-rm.1", page("dock-container-rm", "remove containers"))
	// remote-ext is a command of dock itself: dock-remote(1) never mentions it.
	write("dock-remote.1", page("dock-remote", "manage remotes"))
	write("dock-remote-ext.1", page("dock-remote-ext", "external transport"))

	top := &Spec{Command: "dock", Source: "man", Page: filepath.Join(dir, "dock.1")}
	if got := names(manSubcommands(t.Context(), top, false)); !reflect.DeepEqual(got, []string{"container", "remote", "remote-ext", "run"}) {
		t.Errorf("dock = %v", got)
	}
	group := &Spec{Command: "dock container", Source: "man", Page: filepath.Join(dir, "dock-container.1")}
	subs := manSubcommands(t.Context(), group, false)
	if got := names(subs); !reflect.DeepEqual(got, []string{"ls", "rm"}) || subs[0].Desc != "list containers" {
		t.Errorf("dock container = %+v", subs)
	}
	remote := &Spec{Command: "dock remote", Source: "man", Page: filepath.Join(dir, "dock-remote.1")}
	if got := manSubcommands(t.Context(), remote, false); len(got) != 0 {
		t.Errorf("dock remote = %v", got)
	}
}

// A tool documented only by --help, with a command group two levels deep.
func TestLoadLineNestedHelp(t *testing.T) {
	t.Setenv("COMMANDO_CACHE_DIR", t.TempDir())
	t.Setenv("COMMANDO_NO_COMPLETIONS", "1")
	bin := t.TempDir()
	script := `#!/bin/sh
case "$*" in
"container ls --help") printf 'Usage:  fakedock container ls [OPTIONS]\n\nList containers\n\nOptions:\n  -a, --all     Show all containers\n  -q, --quiet   Only display IDs\n' ;;
"container --help") printf 'Usage:  fakedock container COMMAND\n\nManage containers\n\nCommands:\n  ls          List containers\n  rm          Remove containers\n' ;;
*) printf 'Usage:  fakedock [OPTIONS] COMMAND\n\nOptions:\n  -v, --version   Print version\n  -D, --debug     Debug mode\n\nCommands:\n  container   Manage containers\n  run         Run a container\n' ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "fakedock"), []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Chdir(t.TempDir())

	for _, tc := range []struct {
		words []string
		cmd   string
		n     int
		subs  []string
	}{
		{[]string{"fakedock"}, "fakedock", 1, []string{"container", "run"}},
		{[]string{"fakedock", "container"}, "fakedock container", 2, []string{"ls", "rm"}},
		{[]string{"fakedock", "container", "ls", "-a"}, "fakedock container ls", 3, nil},
		{[]string{"fakedock", "image"}, "fakedock", 1, []string{"container", "run"}},
	} {
		s, n, err := LoadLine(t.Context(), tc.words, false)
		if err != nil {
			t.Fatalf("%v: %v", tc.words, err)
		}
		if s.Command != tc.cmd || n != tc.n {
			t.Errorf("%v = %q, %d; want %q, %d", tc.words, s.Command, n, tc.cmd, tc.n)
		}
		if got := names(Subcommands(t.Context(), s, false)); !reflect.DeepEqual(got, tc.subs) {
			t.Errorf("%v subcommands = %v, want %v", tc.words, got, tc.subs)
		}
	}
}
