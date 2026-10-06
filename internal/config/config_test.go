package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/lonedevel/commando/internal/manpage"
)

func write(t *testing.T, text string) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "config.toml")
	if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	return p
}

func TestLoad(t *testing.T) {
	// No file: the defaults.
	c, err := LoadFile(filepath.Join(t.TempDir(), "missing.toml"))
	if err != nil || !reflect.DeepEqual(c, Default()) {
		t.Fatalf("missing = %+v, %v", c, err)
	}

	c, err = LoadFile(write(t, `
long = true
recent = 25
theme = "light"
[colors]
danger = "#FF0000"
[commands."git push"]
safe = ["--force-with-lease"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if !c.Long || !c.Confirm || !c.History || c.Recent != 25 || c.Theme != "light" || c.Colors["danger"] != "#FF0000" ||
		!reflect.DeepEqual(c.Commands["git push"].Safe, []string{"--force-with-lease"}) {
		t.Errorf("loaded %+v", c)
	}

	// Problems are reported, and the rest of the file still applies.
	c, err = LoadFile(write(t, "history = false\ntheme = \"neon\"\nbogus = 1\n[colors]\nsparkle = \"#fff\"\n"))
	if err == nil || !strings.Contains(err.Error(), "bogus") || !strings.Contains(err.Error(), `theme "neon"`) || !strings.Contains(err.Error(), `"sparkle"`) {
		t.Errorf("err = %v", err)
	}
	if c.History || c.Theme != "auto" || len(c.Colors) != 0 {
		t.Errorf("after problems: %+v", c)
	}

	// Broken TOML: the defaults, and the error.
	if c, err := LoadFile(write(t, "long = \n")); err == nil || !reflect.DeepEqual(c, Default()) {
		t.Errorf("broken = %+v, %v", c, err)
	}
}

func TestStarter(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "sub", "config.toml")
	if wrote, err := WriteStarter(p); !wrote || err != nil {
		t.Fatalf("WriteStarter = %v, %v", wrote, err)
	}
	if wrote, _ := WriteStarter(p); wrote {
		t.Error("overwrote an existing file")
	}
	if c, err := LoadFile(p); err != nil || !reflect.DeepEqual(c, Default()) {
		t.Errorf("starter = %+v, %v", c, err)
	}
	// Every example in it is valid once uncommented.
	var lines []string
	for _, l := range strings.Split(Starter, "\n") {
		t := strings.TrimPrefix(l, "# ")
		if strings.Contains(t, " = ") || strings.HasPrefix(t, "[") {
			lines = append(lines, t)
		}
	}
	if _, err := LoadFile(write(t, strings.Join(lines, "\n"))); err != nil {
		t.Errorf("uncommented starter: %v\n%s", err, strings.Join(lines, "\n"))
	}
}

func TestApply(t *testing.T) {
	c := &Config{Commands: map[string]Command{"ls": {
		Safe:   []string{"-f"},
		Risky:  []string{"-a"},
		Values: map[string][]string{"--quoting-style": {"clocale"}, "--width": {"80"}, "--color": {"auto"}},
	}}}
	s := &manpage.Spec{Command: "ls", Options: []manpage.Option{
		{Names: []string{"-f"}, Danger: "can delete or overwrite data"},
		{Names: []string{"-a", "--all"}},
		{Names: []string{"--quoting-style"}, Arg: "WORD", Kind: manpage.KindChoice, Choices: []string{"literal", "c"}},
		{Names: []string{"--width"}, Arg: "COLS", Kind: manpage.KindNumber},
		{Names: []string{"--color"}, Arg: "WHEN", Kind: manpage.KindString},
	}}
	c.Apply(s)
	if s.Options[0].Danger != "" || s.Options[1].Danger == "" {
		t.Errorf("danger: %q %q", s.Options[0].Danger, s.Options[1].Danger)
	}
	if got := s.Options[2].Choices; !reflect.DeepEqual(got, []string{"literal", "c", "clocale"}) {
		t.Errorf("--quoting-style = %v", got)
	}
	if s.Options[3].Kind != manpage.KindNumber {
		t.Error("a number option became a dropdown")
	}
	if o := s.Options[4]; o.Kind != manpage.KindChoice || !reflect.DeepEqual(o.Choices, []string{"auto"}) {
		t.Errorf("--color = %v %v", o.Kind, o.Choices)
	}
	// Other commands, and no settings at all, are left alone.
	other := &manpage.Spec{Command: "cp", Options: []manpage.Option{{Names: []string{"-f"}, Danger: "x"}}}
	c.Apply(other)
	(*Config)(nil).Apply(other)
	if other.Options[0].Danger != "x" {
		t.Error("cp was changed")
	}
}

func TestPath(t *testing.T) {
	t.Setenv("COMMANDO_CONFIG", "")
	t.Setenv("XDG_CONFIG_HOME", "/x")
	if p := Path(); p != "/x/commando/config.toml" {
		t.Errorf("Path = %q", p)
	}
	t.Setenv("COMMANDO_CONFIG", "/y.toml")
	if p := Path(); p != "/y.toml" {
		t.Errorf("Path = %q", p)
	}
}
