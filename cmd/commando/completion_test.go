package main

import (
	"os"
	"strings"
	"testing"
)

// Every flag is in the help, the man page and each shell's completions.
func TestFlagsDocumented(t *testing.T) {
	page, err := os.ReadFile("../../docs/commando.1")
	if err != nil {
		t.Fatal(err)
	}
	man := strings.ReplaceAll(string(page), `\-`, "-")
	scripts := map[string]string{}
	for _, sh := range []string{"zsh", "bash", "fish"} {
		s, ok := completion(sh)
		if !ok {
			t.Fatalf("no %s completion", sh)
		}
		scripts[sh] = s
	}
	for _, f := range flagHelp {
		long := "--" + f.long
		if !strings.Contains(usage, long) {
			t.Errorf("%s missing from usage", long)
		}
		if !strings.Contains(man, long) {
			t.Errorf("%s missing from the man page", long)
		}
		if !strings.Contains(scripts["zsh"], long+"[") && !strings.Contains(scripts["zsh"], ","+long+"}") {
			t.Errorf("%s missing from zsh completion", long)
		}
		if !strings.Contains(scripts["bash"], " "+long) && !strings.Contains(scripts["bash"], `"`+long) {
			t.Errorf("%s missing from bash completion", long)
		}
		if !strings.Contains(scripts["fish"], " -l "+f.long+" ") {
			t.Errorf("%s missing from fish completion", long)
		}
	}
	// And the other way: every flag in the help has a completion.
	known := map[string]bool{}
	for _, f := range flagHelp {
		known["--"+f.long] = true
	}
	flags := usage[strings.Index(usage, "Flags:"):strings.Index(usage, "Examples:")]
	for _, w := range strings.Fields(flags) {
		if w = strings.TrimSuffix(w, ","); strings.HasPrefix(w, "--") && !known[w] {
			t.Errorf("%s has no completion", w)
		}
	}
	if _, ok := completion("tcsh"); ok {
		t.Error("tcsh completion")
	}
}

// The flag parser accepts every flag the completions offer.
func TestCompletionFlag(t *testing.T) {
	if code := run([]string{"--completion", "tcsh"}); code != 2 {
		t.Errorf("--completion tcsh = %d", code)
	}
}
