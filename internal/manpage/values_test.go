package manpage

import (
	"os"
	"reflect"
	"strings"
	"testing"
)

func optionNamed(t *testing.T, s *Spec, name string) *Option {
	t.Helper()
	for i := range s.Options {
		for _, n := range s.Options[i].Names {
			if n == name {
				return &s.Options[i]
			}
		}
	}
	t.Fatalf("no option %s", name)
	return nil
}

func TestValueDescsFixtures(t *testing.T) {
	parse := func(file, cmd string) *Spec {
		b, err := os.ReadFile("testdata/" + file + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		return Parse(cmd, Clean(string(b)), "man")
	}
	// macOS find: a list of types under -type, which had no dropdown before.
	o := optionNamed(t, parse("bsd-find", "find"), "-type")
	if o.Kind != KindChoice || !reflect.DeepEqual(o.Choices, []string{"b", "c", "d", "f", "l", "p", "s"}) ||
		o.ChoiceDesc["d"] != "Directory" || o.ChoiceDesc["l"] != "Symbolic link" {
		t.Errorf("bsd find -type = %v %v %v", o.Kind, o.Choices, o.ChoiceDesc)
	}
	// …but -atime's list is of unit suffixes, not values.
	if o := optionNamed(t, parse("bsd-find", "find"), "-atime"); o.Kind == KindChoice {
		t.Errorf("-atime became a choice: %v", o.Choices)
	}
	// macOS grep: "binary (default)" with its meaning on the next lines.
	o = optionNamed(t, parse("bsd-grep", "grep"), "--binary-files")
	if o.Kind != KindChoice || o.ChoiceDesc["binary"] != "Search binary files but do not print them" ||
		o.ChoiceDesc["text"] != "Treat all files as text" {
		t.Errorf("bsd grep --binary-files = %v %v", o.Choices, o.ChoiceDesc)
	}
	// GNU ls: "none (-U), size (-S)" means the same as those flags.
	o = optionNamed(t, parse("gnu-ls", "ls"), "--sort")
	if d := o.ChoiceDesc["size"]; !strings.HasPrefix(d, "Like -S: ") {
		t.Errorf("ls --sort size = %q", d)
	}
	// "access time (-u): atime, access, use; …"
	o = optionNamed(t, parse("gnu-ls", "ls"), "--time")
	if d := o.ChoiceDesc["atime"]; d != "Access time (like -u)" {
		t.Errorf("ls --time atime = %q", d)
	}
	// GNU tar: environment variables listed under --to-command aren't values.
	if o := optionNamed(t, parse("gnu-tar", "tar"), "--to-command"); o.Kind == KindChoice {
		t.Errorf("--to-command became a choice: %v", o.Choices)
	}
}

func TestDefinitionListLayouts(t *testing.T) {
	page := `NAME
       tool - does things

OPTIONS
       --format=FORMAT
              Create archive of the given format.  Valid formats are:

              gnu    GNU tar 1.13.x format

              oldgnu GNU format as per tar <= 1.12.

              pax, posix
                     POSIX 1003.1-2001 (pax) format.

              v7     Old V7 tar format.

       --date=FORMAT
              How to show dates.

              --date=relative shows dates relative to the current time, e.g. "2 hours ago".

              --date=iso (or --date=iso8601) shows timestamps in a ISO 8601-like format.

       --binary-files=TYPE
              What to do with binary files.

              If TYPE is without-match, grep assumes the rest of the file does not match.

              If TYPE is text, grep processes a binary file as if it were text.
`
	s := Parse("tool", Clean(page), "man")
	o := optionNamed(t, s, "--format")
	want := map[string]string{
		"gnu": "GNU tar 1.13.x format", "oldgnu": "GNU format as per tar <= 1.12",
		"pax": "POSIX 1003.1-2001 (pax) format", "posix": "POSIX 1003.1-2001 (pax) format",
		"v7": "Old V7 tar format",
	}
	if o.Kind != KindChoice || !reflect.DeepEqual(o.ChoiceDesc, want) {
		t.Errorf("--format = %v %v", o.Choices, o.ChoiceDesc)
	}
	o = optionNamed(t, s, "--date")
	if o.Kind != KindChoice || !reflect.DeepEqual(o.Choices, []string{"relative", "iso", "iso8601"}) ||
		o.ChoiceDesc["iso8601"] != "Shows timestamps in a ISO 8601-like format" {
		t.Errorf("--date = %v %v", o.Choices, o.ChoiceDesc)
	}
	o = optionNamed(t, s, "--binary-files")
	if o.ChoiceDesc["text"] != "Grep processes a binary file as if it were text" {
		t.Errorf("--binary-files = %v %v", o.Choices, o.ChoiceDesc)
	}
}
