// Package store keeps commando's saved presets and recently run commands.
//
// Everything lives in one small JSON file (see Path). Reads tolerate a
// missing or corrupt file; writes replace it atomically.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// MaxRecent is how many recent command lines are kept per command, unless
// Store.Max says otherwise.
const MaxRecent = 10

// Entry is a saved command line.
type Entry struct {
	Name string    `json:"name,omitempty"` // presets only
	Line string    `json:"line"`           // full command line, e.g. "ls -al ~/src"
	Used time.Time `json:"used"`           // last run (recent) or saved (preset)
}

// Command holds the presets and history for one command ("ls", "git commit").
type Command struct {
	Presets []Entry `json:"presets,omitempty"`
	Recent  []Entry `json:"recent,omitempty"`
}

// Store is the whole file.
type Store struct {
	Version  int                 `json:"version"`
	Commands map[string]*Command `json:"commands"`

	// Max is how many recent lines to keep per command; 0 means MaxRecent.
	Max int `json:"-"`

	path string
}

// Path returns the store file location: $COMMANDO_DATA_DIR/store.json, or
// commando/store.json under the user config directory
// (~/Library/Application Support on macOS, ~/.config on Linux).
func Path() (string, error) {
	if d := os.Getenv("COMMANDO_DATA_DIR"); d != "" {
		return filepath.Join(d, "store.json"), nil
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "commando", "store.json"), nil
}

// Load reads the store. A missing or unreadable file yields an empty store
// that can still be saved.
func Load() *Store {
	s := &Store{Version: 1, Commands: map[string]*Command{}}
	p, err := Path()
	if err != nil {
		return s
	}
	s.path = p
	b, err := os.ReadFile(p)
	if err != nil {
		return s
	}
	var disk Store
	if json.Unmarshal(b, &disk) == nil && disk.Commands != nil {
		s.Commands = disk.Commands
	}
	return s
}

// Save writes the store atomically.
func (s *Store) Save() error {
	if s.path == "" {
		return errors.New("no location for the store file")
	}
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func (s *Store) get(cmd string) *Command {
	c := s.Commands[cmd]
	if c == nil {
		c = &Command{}
		s.Commands[cmd] = c
	}
	return c
}

// For returns the presets (by name) and recent lines (newest first) for cmd.
func (s *Store) For(cmd string) (presets, recent []Entry) {
	c := s.Commands[cmd]
	if c == nil {
		return nil, nil
	}
	presets = append([]Entry(nil), c.Presets...)
	sort.SliceStable(presets, func(i, j int) bool {
		return strings.ToLower(presets[i].Name) < strings.ToLower(presets[j].Name)
	})
	recent = append([]Entry(nil), c.Recent...)
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].Used.After(recent[j].Used) })
	return presets, recent
}

// AddRecent records that line was run for cmd, moving it to the front and
// keeping at most Max (or MaxRecent) entries.
func (s *Store) AddRecent(cmd, line string, now time.Time) {
	line = strings.TrimSpace(line)
	if cmd == "" || line == "" {
		return
	}
	c := s.get(cmd)
	kept := []Entry{{Line: line, Used: now}}
	for _, e := range c.Recent {
		if e.Line != line {
			kept = append(kept, e)
		}
	}
	sort.SliceStable(kept[1:], func(i, j int) bool { return kept[1+i].Used.After(kept[1+j].Used) })
	limit := s.Max
	if limit <= 0 {
		limit = MaxRecent
	}
	if len(kept) > limit {
		kept = kept[:limit]
	}
	c.Recent = kept
}

// SavePreset stores line under name for cmd, replacing a preset of the
// same name (case-insensitive).
func (s *Store) SavePreset(cmd, name, line string, now time.Time) {
	name, line = strings.TrimSpace(name), strings.TrimSpace(line)
	if cmd == "" || name == "" || line == "" {
		return
	}
	c := s.get(cmd)
	for i, p := range c.Presets {
		if strings.EqualFold(p.Name, name) {
			c.Presets[i] = Entry{Name: name, Line: line, Used: now}
			return
		}
	}
	c.Presets = append(c.Presets, Entry{Name: name, Line: line, Used: now})
}

// DeletePreset removes the named preset for cmd.
func (s *Store) DeletePreset(cmd, name string) {
	c := s.Commands[cmd]
	if c == nil {
		return
	}
	out := c.Presets[:0]
	for _, p := range c.Presets {
		if !strings.EqualFold(p.Name, name) {
			out = append(out, p)
		}
	}
	c.Presets = out
	s.prune(cmd)
}

// DeleteRecent removes a line from cmd's history.
func (s *Store) DeleteRecent(cmd, line string) {
	c := s.Commands[cmd]
	if c == nil {
		return
	}
	out := c.Recent[:0]
	for _, e := range c.Recent {
		if e.Line != line {
			out = append(out, e)
		}
	}
	c.Recent = out
	s.prune(cmd)
}

func (s *Store) prune(cmd string) {
	if c := s.Commands[cmd]; c != nil && len(c.Presets) == 0 && len(c.Recent) == 0 {
		delete(s.Commands, cmd)
	}
}

// AllRecent returns the most recent lines across every command, newest
// first, at most n.
func (s *Store) AllRecent(n int) []Entry {
	var all []Entry
	for _, c := range s.Commands {
		all = append(all, c.Recent...)
	}
	sort.SliceStable(all, func(i, j int) bool { return all[i].Used.After(all[j].Used) })
	if len(all) > n {
		all = all[:n]
	}
	return all
}

// Under returns the presets and recent lines saved for cmd and for each of
// its subcommands ("git", "git commit", "git push"…). Presets are sorted by
// name, recent lines newest first, at most n.
func (s *Store) Under(cmd string, n int) (presets, recent []Entry) {
	for key, c := range s.Commands {
		if key != cmd && !strings.HasPrefix(key, cmd+" ") {
			continue
		}
		presets = append(presets, c.Presets...)
		recent = append(recent, c.Recent...)
	}
	sort.SliceStable(presets, func(i, j int) bool {
		return strings.ToLower(presets[i].Name) < strings.ToLower(presets[j].Name)
	})
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].Used.After(recent[j].Used) })
	if len(recent) > n {
		recent = recent[:n]
	}
	return presets, recent
}

// Search returns the saved lines, presets and recent ones of every
// command, that contain all of words (case-insensitive) in the line or the
// preset's name: presets first, then recent lines newest first, at most n.
// A line saved as both is listed once, as the preset.
func (s *Store) Search(words []string, n int) []Entry {
	match := func(e Entry) bool {
		hay := strings.ToLower(e.Line + " " + e.Name)
		for _, w := range words {
			if !strings.Contains(hay, strings.ToLower(w)) {
				return false
			}
		}
		return true
	}
	var presets, recent []Entry
	for _, c := range s.Commands {
		for _, p := range c.Presets {
			if match(p) {
				presets = append(presets, p)
			}
		}
		for _, r := range c.Recent {
			if match(r) {
				recent = append(recent, r)
			}
		}
	}
	sort.SliceStable(presets, func(i, j int) bool {
		return strings.ToLower(presets[i].Name) < strings.ToLower(presets[j].Name)
	})
	sort.SliceStable(recent, func(i, j int) bool { return recent[i].Used.After(recent[j].Used) })
	seen := map[string]bool{}
	var out []Entry
	for _, e := range append(presets, recent...) {
		if seen[e.Line] || len(out) == n {
			continue
		}
		seen[e.Line] = true
		out = append(out, e)
	}
	return out
}
