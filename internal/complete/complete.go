// Package complete reads the value lists that shell completion definitions
// give for command options, e.g. zsh's
//
//	'--color=-[control use of color]:color:(never always auto)'
//
// or fish's
//
//	complete -c ls -l color -a "never always auto"
//
// so options can be offered as dropdowns with the values the shell knows.
package complete

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Values maps an option name ("--color", "-X") to its allowed values.
type Values map[string][]string

// Result is what was found for a command.
type Result struct {
	Values  Values
	Sources map[string]string // per option: "zsh", "fish" or "zsh and fish"
}

var valueRe = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.+@%:/-]*$`)

func validValue(s string) bool { return len(s) <= 40 && valueRe.MatchString(s) }

// Lookup finds completion values for command (a single word such as "ls").
// zsh definitions come first, since macOS always ships them; fish
// definitions add options and values zsh didn't list.
func Lookup(command string) Result {
	if command == "" || strings.ContainsAny(command, " /") {
		return Result{}
	}
	res := Result{Values: Values{}, Sources: map[string]string{}}
	if p := findZsh(command); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			for name, vals := range ParseZsh(string(b)) {
				res.Values[name] = vals
				res.Sources[name] = "zsh"
			}
		}
	}
	if p := findFish(command); p != "" {
		if b, err := os.ReadFile(p); err == nil {
			for name, vals := range ParseFish(string(b), command) {
				merged := union(res.Values[name], vals)
				switch {
				case res.Sources[name] == "":
					res.Sources[name] = "fish"
				case len(merged) > len(res.Values[name]):
					res.Sources[name] = "zsh and fish"
				}
				res.Values[name] = merged
			}
		}
	}
	return res
}

// zshDirs lists folders that hold zsh completion functions. They are
// fixed locations rather than $fpath so commando doesn't have to start a
// shell; COMMANDO_ZSH_COMPLETIONS (colon-separated) adds more.
func zshDirs() []string {
	var dirs []string
	if extra := os.Getenv("COMMANDO_ZSH_COMPLETIONS"); extra != "" {
		dirs = append(dirs, filepath.SplitList(extra)...)
	}
	if home, err := os.UserHomeDir(); err == nil {
		dirs = append(dirs, filepath.Join(home, ".zsh", "completions"), filepath.Join(home, ".zfunc"))
	}
	dirs = append(dirs,
		"/opt/homebrew/share/zsh/site-functions",
		"/usr/local/share/zsh/site-functions",
		"/usr/share/zsh/site-functions",
		"/usr/share/zsh/vendor-completions",
	)
	// macOS: /usr/share/zsh/5.9/functions; Linux: /usr/share/zsh/functions/Completion/*
	for _, pat := range []string{
		"/usr/share/zsh/*/functions",
		"/usr/share/zsh/functions/Completion/*",
		"/opt/homebrew/share/zsh/functions",
		"/usr/local/share/zsh/functions",
	} {
		m, _ := filepath.Glob(pat)
		dirs = append(dirs, m...)
	}
	return dirs
}

func findZsh(command string) string {
	for _, d := range zshDirs() {
		p := filepath.Join(d, "_"+command)
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p
		}
	}
	return ""
}

func fishDirs() []string {
	var dirs []string
	if extra := os.Getenv("COMMANDO_FISH_COMPLETIONS"); extra != "" {
		dirs = append(dirs, filepath.SplitList(extra)...)
	}
	cfg := os.Getenv("XDG_CONFIG_HOME")
	if cfg == "" {
		if home, err := os.UserHomeDir(); err == nil {
			cfg = filepath.Join(home, ".config")
		}
	}
	if cfg != "" {
		dirs = append(dirs, filepath.Join(cfg, "fish", "completions"))
	}
	for _, prefix := range []string{"/opt/homebrew", "/usr/local", "/usr"} {
		dirs = append(dirs,
			filepath.Join(prefix, "share", "fish", "vendor_completions.d"),
			filepath.Join(prefix, "share", "fish", "completions"),
		)
	}
	return dirs
}

func findFish(command string) string {
	for _, d := range fishDirs() {
		p := filepath.Join(d, command+".fish")
		if st, err := os.Stat(p); err == nil && st.Mode().IsRegular() {
			return p
		}
	}
	return ""
}

// union returns a followed by the items of b it lacks.
func union(a, b []string) []string {
	out := append([]string(nil), a...)
	seen := map[string]bool{}
	for _, x := range a {
		seen[x] = true
	}
	for _, x := range b {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	return out
}
