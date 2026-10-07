// Package config reads commando's settings file, config.toml.
package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/BurntSushi/toml"

	"github.com/lonedevel/commando/internal/manpage"
)

// Config is the settings file. Command-line flags and environment
// variables win over it.
type Config struct {
	Long    bool              `toml:"long"`    // prefer --long option names
	Confirm bool              `toml:"confirm"` // ask before running a command with a risky option
	History bool              `toml:"history"` // remember commands, offer presets
	Recent  int               `toml:"recent"`  // recent commands kept per command
	Theme   string            `toml:"theme"`   // auto, dark, light or contrast
	Colors  map[string]string `toml:"colors"`  // overrides, by role: "accent" = "#C084FC"

	// Corrections per command, keyed by its full name ("ls", "git push").
	Commands map[string]Command `toml:"commands"`
}

// Command holds corrections to what commando read from one manual.
type Command struct {
	Safe   []string            `toml:"safe"`   // options never marked ⚠
	Risky  []string            `toml:"risky"`  // options always marked ⚠
	Values map[string][]string `toml:"values"` // extra dropdown values, by option name
}

// Themes are the accepted values of theme: commando's own palette in four
// variants, then named color schemes, meant for a terminal set to the same
// scheme.
var Themes = []string{
	"auto", "dark", "light", "contrast",
	"dracula", "nord", "gruvbox-dark", "gruvbox-light", "solarized-dark", "solarized-light",
	"catppuccin-mocha", "catppuccin-latte", "tokyo-night",
}

// ColorRoles are the names [colors] accepts.
var ColorRoles = []string{"accent", "command", "header", "option", "value", "argument", "path", "group", "danger", "text", "dim", "faint"}

// Default is the configuration without a file.
func Default() *Config {
	return &Config{Confirm: true, History: true, Recent: 10, Theme: "auto"}
}

// Path is where the settings file is read from: $COMMANDO_CONFIG, else
// config.toml in $XDG_CONFIG_HOME/commando or ~/.config/commando.
func Path() string {
	if p := os.Getenv("COMMANDO_CONFIG"); p != "" {
		return p
	}
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "commando", "config.toml")
}

// Load reads the settings file. A missing file gives the defaults; a
// broken one gives the defaults and an error saying what is wrong, so
// commando can warn and carry on.
func Load() (*Config, error) {
	return LoadFile(Path())
}

// LoadFile reads the settings file at path.
func LoadFile(path string) (*Config, error) {
	c := Default()
	if path == "" {
		return c, nil
	}
	b, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return c, nil
	}
	if err != nil {
		return c, err
	}
	md, err := toml.Decode(string(b), c)
	if err != nil {
		return Default(), fmt.Errorf("%s: %w", path, err)
	}
	var problems []string
	if und := md.Undecoded(); len(und) > 0 {
		var keys []string
		for _, k := range und {
			keys = append(keys, k.String())
		}
		problems = append(problems, "unknown setting "+strings.Join(keys, ", "))
	}
	if !contains(Themes, c.Theme) {
		problems = append(problems, fmt.Sprintf("theme %q is not one of %s", c.Theme, strings.Join(Themes, ", ")))
		c.Theme = "auto"
	}
	var roles []string
	for r := range c.Colors {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	for _, r := range roles {
		if !contains(ColorRoles, r) {
			problems = append(problems, fmt.Sprintf("unknown color %q (use %s)", r, strings.Join(ColorRoles, ", ")))
			delete(c.Colors, r)
		}
	}
	if c.Recent < 1 {
		c.Recent = 1
	}
	if len(problems) > 0 {
		return c, fmt.Errorf("%s: %s", path, strings.Join(problems, "; "))
	}
	return c, nil
}

// Apply makes the file's corrections to a command's options.
func (c *Config) Apply(s *manpage.Spec) {
	if c == nil || s == nil {
		return
	}
	cmd, ok := c.Commands[s.Command]
	if !ok {
		return
	}
	for i := range s.Options {
		o := &s.Options[i]
		for _, n := range o.Names {
			if contains(cmd.Safe, n) {
				o.Danger = ""
			}
			if contains(cmd.Risky, n) {
				o.Danger = "marked risky in your config"
			}
			extra := cmd.Values[n]
			if len(extra) == 0 || !o.TakesArg() || (o.Kind != manpage.KindString && o.Kind != manpage.KindChoice) {
				continue
			}
			for _, v := range extra {
				if !contains(o.Choices, v) {
					o.Choices = append(o.Choices, v)
				}
			}
			o.Kind = manpage.KindChoice
		}
	}
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Starter is a commented settings file, written by commando --config.
const Starter = `# commando settings. Flags and environment variables win over these.
# Lines starting with # are comments; remove the # to use a setting.

# Prefer --long option names over short ones (like --long).
# long = false

# Ask before running a command that uses a risky option (⚠).
# confirm = true

# Remember commands you run, and offer presets and recent commands.
# history = true

# How many recent commands to keep for each command.
# recent = 10

# Colors: "auto" follows your terminal's background; or "dark", "light",
# or "contrast" for stronger colors. Or a color scheme, to match a terminal
# that uses it: "dracula", "nord", "gruvbox-dark", "gruvbox-light",
# "solarized-dark", "solarized-light", "catppuccin-mocha",
# "catppuccin-latte" or "tokyo-night".
# theme = "auto"

# Override any color by its role: accent, command, header, option, value,
# argument, path, group, danger, text, dim, faint.
# [colors]
# accent = "#C084FC"
# danger = "#FF5555"

# Corrections for one command, by its full name ("ls", "git push").
# [commands."docker run"]
# safe = ["--rm"]                  # never mark these ⚠
# risky = ["--privileged"]         # always mark these ⚠
#
# [commands.ls.values]             # extra values for a dropdown
# "--quoting-style" = ["clocale"]
`

// WriteStarter writes Starter to path unless a file is already there. It
// reports whether it wrote one.
func WriteStarter(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, err
	}
	return true, os.WriteFile(path, []byte(Starter), 0o644)
}
