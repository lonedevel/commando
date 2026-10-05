package manpage

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"
)

// parserVersion is part of the cache key; bump it when parsing changes.
const parserVersion = "6"

// RenderWidth is the column width manual pages are rendered at.
const RenderWidth = 100

// ErrNotFound means neither a manual page nor --help output was available.
var ErrNotFound = errors.New("no manual entry or --help output")

// Locate returns the manual page file for name ("git-commit", "ls"), or "".
func Locate(ctx context.Context, name string) string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "man", "-w", name).Output()
	if err != nil {
		return ""
	}
	first := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if first == "" {
		return ""
	}
	if _, err := os.Stat(first); err != nil {
		return ""
	}
	return first
}

// Load returns the Spec for a command, using the on-disk cache when the
// underlying manual page (or binary, for --help) has not changed.
// words is the command plus any subcommand, e.g. ["git", "commit"].
func Load(ctx context.Context, words []string, useCache bool) (*Spec, error) {
	if len(words) == 0 {
		return nil, ErrNotFound
	}
	display := strings.Join(words, " ")
	page := strings.Join(words, "-")

	if path := Locate(ctx, page); path != "" {
		key := cacheKey("man", page, path)
		if useCache {
			if s := readCache(key); s != nil {
				return s, nil
			}
		}
		text, err := renderMan(ctx, page)
		if err == nil && strings.Count(text, "\n") > 3 {
			s := Parse(display, Clean(text), "man")
			if len(s.Options) > 0 || s.Summary != "" {
				writeCache(key, s)
				return s, nil
			}
		}
	}

	bin, err := exec.LookPath(words[0])
	if err != nil {
		return nil, fmt.Errorf("%s: %w", display, ErrNotFound)
	}
	key := cacheKey("help", display, bin)
	if useCache {
		if s := readCache(key); s != nil {
			return s, nil
		}
	}
	if len(words) > 1 {
		// Only probe subcommands through the subcommand itself.
		if text := runHelp(ctx, bin, words[1:], "--help"); text != "" {
			s := Parse(display, Clean(text), "help")
			if len(s.Options) > 0 {
				writeCache(key, s)
				return s, nil
			}
		}
		return nil, fmt.Errorf("%s: %w", display, ErrNotFound)
	}
	var fallback *Spec
	for _, flag := range []string{"--help", "-h"} {
		text := runHelp(ctx, bin, words[1:], flag)
		if strings.TrimSpace(text) == "" {
			continue
		}
		s := Parse(display, Clean(text), "help")
		if len(s.Options) > 0 {
			writeCache(key, s)
			return s, nil
		}
		if fallback == nil && strings.Count(text, "\n") >= 2 {
			fallback = s
		}
	}
	if fallback != nil {
		// Documented, but no recognizable options: still offer the form
		// for arguments, with the help text as the manual.
		return fallback, nil
	}
	return nil, fmt.Errorf("%s: %w", display, ErrNotFound)
}

func renderMan(ctx context.Context, page string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "man", page)
	w := fmt.Sprint(RenderWidth)
	cmd.Env = append(os.Environ(),
		"MANPAGER=cat", "PAGER=cat", "MANWIDTH="+w, "COLUMNS="+w,
		"GROFF_NO_SGR=1", "MAN_KEEP_FORMATTING=",
	)
	if runtime.GOOS != "darwin" {
		// man-db: no hyphenation or justification, so text joins cleanly.
		cmd.Env = append(cmd.Env, "MANOPT=--no-hyphenation --no-justification")
	}
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil && out.Len() == 0 {
		return "", err
	}
	return out.String(), nil
}

func runHelp(ctx context.Context, bin string, sub []string, flag string) string {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	args := append(append([]string{}, sub...), flag)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Env = append(os.Environ(), "COLUMNS=100", "PAGER=cat", "NO_COLOR=1", "TERM=dumb")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	cmd.Stdin = nil
	_ = cmd.Run()
	if ctx.Err() != nil {
		return ""
	}
	return out.String()
}

func cacheDir() string {
	if d := os.Getenv("COMMANDO_CACHE_DIR"); d != "" {
		return d
	}
	d, err := os.UserCacheDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "commando")
}

func cacheKey(kind, name, path string) string {
	h := sha1.New()
	fmt.Fprintf(h, "%s|%s|%s|%s|%d", parserVersion, kind, name, path, RenderWidth)
	if st, err := os.Stat(path); err == nil {
		fmt.Fprintf(h, "|%d|%d", st.Size(), st.ModTime().UnixNano())
	}
	return hex.EncodeToString(h.Sum(nil))[:20]
}

func readCache(key string) *Spec {
	dir := cacheDir()
	if dir == "" {
		return nil
	}
	b, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		return nil
	}
	var s Spec
	if json.Unmarshal(b, &s) != nil {
		return nil
	}
	return &s
}

func writeCache(key string, s *Spec) {
	dir := cacheDir()
	if dir == "" {
		return
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return
	}
	b, err := json.Marshal(s)
	if err != nil {
		return
	}
	tmp := filepath.Join(dir, key+".tmp")
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, filepath.Join(dir, key+".json"))
	}
}

var subcmdRe = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

// LoadLine loads the spec for a command line given as words (command first,
// then its existing arguments). It recognizes subcommands such as
// "git commit" (git-commit(1)) or "cargo build" (cargo build --help) and
// reports how many words name the command.
func LoadLine(ctx context.Context, words []string, useCache bool) (*Spec, int, error) {
	if len(words) == 0 {
		return nil, 0, ErrNotFound
	}
	sub := len(words) > 1 && subcmdRe.MatchString(words[1])
	if sub {
		if _, err := os.Stat(words[1]); err == nil {
			sub = false // an existing file, not a subcommand
		}
	}
	if sub && HasPage(ctx, words[0]+"-"+words[1]) {
		if s, err := Load(ctx, words[:2], useCache); err == nil {
			ApplyDanger(s)
			return s, 2, nil
		}
	}
	s, err := Load(ctx, words[:1], useCache)
	if err != nil {
		return nil, 0, err
	}
	// Tools documented only by --help: try "cmd sub --help" when the
	// top-level help lists sub as a command.
	if sub && s.Source == "help" && listsCommand(s.Manual, words[1]) {
		if ss, err := Load(ctx, words[:2], useCache); err == nil && ss.Source == "help" &&
			strings.Contains(strings.ToLower(ss.Synopsis+" "+ss.Manual), strings.ToLower(words[0]+" "+words[1])) {
			ApplyDanger(ss)
			return ss, 2, nil
		}
	}
	ApplyCompletions(s, completionsFor(words[:1]))
	ApplyDanger(s)
	return s, 1, nil
}

func listsCommand(help, sub string) bool {
	re := regexp.MustCompile(`(?m)^\s+` + regexp.QuoteMeta(sub) + `(?:,\s*\S+)*(?:\s{2,}|$)`)
	return re.MatchString(help)
}

// HasPage reports whether a manual page exists for name.
func HasPage(ctx context.Context, name string) bool { return Locate(ctx, name) != "" }
