package manpage

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func examplesOf(t *testing.T, command, text, source string) []Example {
	t.Helper()
	return Parse(command, Clean(text), source).Examples
}

func TestExamplesFixtures(t *testing.T) {
	read := func(name string) string {
		b, err := os.ReadFile("testdata/" + name + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	// BSD find: the explanation is indented below each command.
	ex := examplesOf(t, "find", read("bsd-find"), "man")
	if len(ex) < 5 || ex[0].Line != `find / \! -name "*.c" -print` ||
		ex[0].Desc != "Print out a list of all the files whose names do not end in .c." {
		t.Errorf("bsd find = %+v", ex)
	}
	// BSD grep: a bulleted explanation before each "$ grep …" line; the
	// examples that pipe (egrep … | …, echo … | grep) are left out.
	ex = examplesOf(t, "grep", read("bsd-grep"), "man")
	if len(ex) < 6 || ex[0].Line != "grep 'patricia' myfile" || ex[0].Desc != "Find all occurrences of the pattern ‘patricia’ in a file" {
		t.Fatalf("bsd grep = %+v", ex)
	}
	for _, e := range ex {
		if hasShellOperator(e.Line) || !strings.HasPrefix(e.Line, "grep ") {
			t.Errorf("kept %q", e.Line)
		}
	}
	if e := ex[4]; e.Line != `grep '^\.Pp' myfile` || !strings.HasPrefix(e.Desc, "Find all occurrences of the pattern ‘.Pp’") {
		t.Errorf("ex[4] = %+v", e)
	}
	// No EXAMPLES section: none.
	if ex := examplesOf(t, "ls", read("gnu-ls"), "man"); ex != nil {
		t.Errorf("gnu ls = %+v", ex)
	}
}

func TestExamplesLayouts(t *testing.T) {
	gitLog := `NAME
       git-log - Show commit logs

EXAMPLES
       git log --no-merges
           Show the whole commit history, but skip any merges

       git log --since="2 weeks ago" -- gitk
           Show the changes during the last two weeks to the file gitk.

       git log -3
           Limits the number of commits to show to 3.

       git log | less
           Pipes are left out.
`
	want := []Example{
		{`git log --no-merges`, "Show the whole commit history, but skip any merges"},
		{`git log --since="2 weeks ago" -- gitk`, "Show the changes during the last two weeks to the file gitk."},
		{`git log -3`, "Limits the number of commits to show to 3."},
	}
	if got := examplesOf(t, "git log", gitLog, "man"); !reflect.DeepEqual(got, want) {
		t.Errorf("git log =\n%+v", got)
	}

	// rsync: commands first, explained by the paragraph after them; two in
	// a row share one explanation, and a paragraph ending in ":" leads into
	// the next example.
	rsync := `USAGE
       Perhaps the best way to explain the syntax is with some examples:

           rsync -t *.c foo:src/

       This would transfer all files matching the pattern *.c.

       Each of the following commands copies the files in the same way:

           rsync -av /src/foo /dest
           rsync -av /src/foo/ /dest/foo

       To move some files from a remote host, you could run:

           rsync -aiv --remove-source-files rhost:/tmp/a.c \
             ~/src/
`
	want = []Example{
		{"rsync -t *.c foo:src/", "This would transfer all files matching the pattern *.c."},
		{"rsync -av /src/foo /dest", "Each of the following commands copies the files in the same way"},
		{"rsync -av /src/foo/ /dest/foo", "Each of the following commands copies the files in the same way"},
		{"rsync -aiv --remove-source-files rhost:/tmp/a.c ~/src/", "To move some files from a remote host, you could run"},
	}
	if got := examplesOf(t, "rsync", rsync, "man"); !reflect.DeepEqual(got, want) {
		t.Errorf("rsync =\n%+v", got)
	}

	// --help with "# comment" lines above each command.
	help := `Display one or many resources.

Examples:
  # List all pods in ps output format
  kubectl get pods

  # List a single replication controller with specified NAME in JSON output format
  kubectl get -o json replicationcontroller web

  # Pipes are left out
  kubectl get pods -o name | head

Options:
  -A, --all-namespaces=false: List across all namespaces.
`
	want = []Example{
		{"kubectl get pods", "List all pods in ps output format"},
		{"kubectl get -o json replicationcontroller web", "List a single replication controller with specified NAME in JSON output format"},
	}
	if got := examplesOf(t, "kubectl get", help, "help"); !reflect.DeepEqual(got, want) {
		t.Errorf("kubectl get =\n%+v", got)
	}

	// Prose starting with the command's name isn't an example.
	prose := "EXAMPLES\n       find is often used like this, with many options and no dashes at all.\n\n       find . -name core\n           Find core files.\n"
	if got := examplesOf(t, "find", prose, "man"); len(got) != 1 || got[0].Line != "find . -name core" {
		t.Errorf("prose = %+v", got)
	}
}

func TestHasShellOperator(t *testing.T) {
	for s, want := range map[string]bool{
		`find . -exec file '{}' \;`:    false,
		`grep 'a|b' file`:              false,
		`grep "x;y" file`:              false,
		`find . | xargs rm`:            true,
		`ls > out.txt`:                 true,
		`echo $(date)`:                 true,
		`a && b`:                       true,
		"grep `cat pattern` file":      true,
		`find / \( -perm -4000 \) -ls`: false,
	} {
		if got := hasShellOperator(s); got != want {
			t.Errorf("hasShellOperator(%q) = %v", s, got)
		}
	}
}
