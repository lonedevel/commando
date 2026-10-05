package complete

import (
	"os"
	"reflect"
	"testing"
)

func read(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestParseZsh(t *testing.T) {
	v := ParseZsh(read(t, "testdata/zsh/_frob"))
	want := Values{
		"--color":       {"never", "always", "auto"},
		"--sort":        {"size", "time", "none"},
		"--directories": {"read", "skip", "recurse"},
		"-d":            {"read", "skip", "recurse"},
		"--colour":      {"always", "never"},
		"--tint":        {"always", "never"},
		"--quoting":     {"literal", "shell", "c"},
		"--format":      {"gnu", "pax", "ustar"},
		"--level":       {"1", "2", "3"},
	}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("ParseZsh =\n%v\nwant\n%v", v, want)
	}
}

func TestParseFish(t *testing.T) {
	v := ParseFish(read(t, "testdata/fish/frob.fish"), "frob")
	want := Values{
		"--color":    {"never", "always", "auto"},
		"-X":         {"GET", "POST", "PATCH"},
		"--request":  {"GET", "POST", "PATCH"},
		"--sort":     {"size", "extension", "none"},
		"--key-type": {"PEM", "DER", "ENG"},
		"-depth":     {"1", "2"},
	}
	if !reflect.DeepEqual(v, want) {
		t.Errorf("ParseFish =\n%v\nwant\n%v", v, want)
	}
}

func TestLookupMergesShells(t *testing.T) {
	t.Setenv("COMMANDO_ZSH_COMPLETIONS", "testdata/zsh")
	t.Setenv("COMMANDO_FISH_COMPLETIONS", "testdata/fish")
	r := Lookup("frob")
	if got := r.Values["--sort"]; !reflect.DeepEqual(got, []string{"size", "time", "none", "extension"}) {
		t.Errorf("--sort = %v", got)
	}
	for name, src := range map[string]string{"--sort": "zsh and fish", "--color": "zsh", "-X": "fish", "--level": "zsh"} {
		if r.Sources[name] != src {
			t.Errorf("source of %s = %q, want %q", name, r.Sources[name], src)
		}
	}
	if r := Lookup("no-such-command-xyz"); len(r.Values) != 0 {
		t.Errorf("unknown command: %v", r.Values)
	}
	if r := Lookup("../etc/passwd"); len(r.Values) != 0 {
		t.Error("path-like command names must be ignored")
	}
}
