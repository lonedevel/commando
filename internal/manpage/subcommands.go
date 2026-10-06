package manpage

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Subcommand is one command of a tool such as git, cargo or docker.
type Subcommand struct {
	Name    string   `json:"name"`
	Aliases []string `json:"aliases,omitempty"`
	Desc    string   `json:"desc,omitempty"`
	Group   string   `json:"group,omitempty"` // heading it was listed under, e.g. "Management Commands"
}

// Subcommands lists the commands of the tool or command s documents, or
// nil when it has none. Tools documented by man pages have one page per
// command (git-commit(1) beside git(1), docker-container-ls(1) beside
// docker-container(1)); the tool's own page orders and groups them. Tools
// documented by --help list them under "Commands:" headings, at any depth
// (docker container --help).
func Subcommands(ctx context.Context, s *Spec, useCache bool) []Subcommand {
	if s == nil {
		return nil
	}
	var subs []Subcommand
	switch {
	case s.Source == "man" && s.Page != "":
		subs = manSubcommands(ctx, s, useCache)
	case s.Source == "help":
		subs = helpSubcommands(s.Manual)
	}
	if len(subs) < 2 {
		return nil
	}
	return subs
}

// manSubcommands finds the pages named after the command (git-*, or
// docker-container-* for "docker container") in the folder of its own page.
func manSubcommands(ctx context.Context, s *Spec, useCache bool) []Subcommand {
	dir := filepath.Dir(s.Page)
	key := cacheKey("subs", s.Command, dir)
	if useCache {
		if subs, ok := readSubsCache(key); ok {
			return subs
		}
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	words := strings.Fields(s.Command)
	page := strings.Join(words, "-")
	prefix := page + "-"
	pages := map[string]string{} // subcommand → page file
	for _, e := range ents {
		n := e.Name()
		if !strings.HasPrefix(n, prefix) {
			continue
		}
		name := pageName(n)[len(prefix):]
		if subcmdRe.MatchString(name) {
			pages[name] = filepath.Join(dir, n)
		}
	}
	// Below the top level, pages such as git-commit-tree(1) belong to the
	// parent ("git commit-tree"), not to "git commit".
	if len(words) > 1 && len(pages) > 0 {
		last := words[len(words)-1]
		if parent, err := Load(ctx, words[:len(words)-1], useCache); err == nil {
			for _, p := range Subcommands(ctx, parent, useCache) {
				if rest, ok := strings.CutPrefix(p.Name, last+"-"); ok {
					delete(pages, rest)
				}
			}
		}
		// And a command's own page names the commands it groups
		// (docker-container(1) lists docker-container-ls(1)); git-remote(1)
		// never mentions git-remote-ext(1), a command of git itself.
		src := pageSource(s.Page)
		for name := range pages {
			if !strings.Contains(src, prefix+name) {
				delete(pages, name)
			}
		}
	}
	var subs []Subcommand
	seen := map[string]bool{}
	add := func(name, group string) {
		if seen[name] {
			return
		}
		seen[name] = true
		subs = append(subs, Subcommand{Name: name, Desc: pageSummary(pages[name], prefix+name), Group: group})
	}
	// The tool's page usually lists its commands by group, most useful first.
	for _, r := range manRefs(s.Manual, page) {
		if _, ok := pages[r.Name]; ok {
			add(r.Name, r.Group)
		}
	}
	var rest []string
	srcs := map[string]string{}
	for name := range pages {
		if !seen[name] && !nestedPage(prefix, name, pages, srcs) {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	group := ""
	if len(subs) > 0 {
		group = "Other commands"
	}
	for _, name := range rest {
		add(name, group)
	}
	writeSubsCache(key, subs)
	return subs
}

// nestedPage reports whether name ("container-ls") belongs to another
// command on the list ("container") rather than being one itself: that
// command's page must mention it.
func nestedPage(prefix, name string, pages, srcs map[string]string) bool {
	for i := strings.IndexByte(name, '-'); i > 0; i = nextDash(name, i) {
		parent, ok := pages[name[:i]]
		if !ok {
			continue
		}
		src, ok := srcs[parent]
		if !ok {
			src = pageSource(parent)
			srcs[parent] = src
		}
		if strings.Contains(src, prefix+name) {
			return true
		}
	}
	return false
}

func nextDash(s string, i int) int {
	j := strings.IndexByte(s[i+1:], '-')
	if j < 0 {
		return -1
	}
	return i + 1 + j
}

// pageName strips the section and compression suffixes: "git-add.1.gz" → "git-add".
func pageName(file string) string {
	for _, ext := range []string{".gz", ".bz2", ".xz", ".zst"} {
		file = strings.TrimSuffix(file, ext)
	}
	if i := strings.LastIndexByte(file, '.'); i > 0 && i+1 < len(file) && file[i+1] >= '0' && file[i+1] <= '9' {
		file = file[:i]
	}
	return file
}

type manRef struct{ Name, Group string }

var titleSmall = map[string]bool{"and": true, "or": true, "of": true, "the": true, "with": true, "to": true, "a": true}

// manRefs finds lines holding only a reference such as "git-add(1)" and the
// heading each falls under.
func manRefs(manual, cmd string) []manRef {
	re := regexp.MustCompile(`^(\s+)` + regexp.QuoteMeta(cmd) + `-([a-z][a-z0-9-]*)\(\d\w*\)\s*$`)
	var refs []manRef
	var heads []string // headings by indentation, outermost first
	var indents []int
	for _, l := range strings.Split(manual, "\n") {
		t := strings.TrimSpace(l)
		if t == "" {
			continue
		}
		ind := len(l) - len(strings.TrimLeft(l, " \t"))
		if m := re.FindStringSubmatch(l); m != nil {
			group := ""
			for k := len(indents) - 1; k >= 0; k-- {
				if indents[k] < ind {
					group = heads[k]
					break
				}
			}
			refs = append(refs, manRef{Name: m[2], Group: group})
			continue
		}
		// Headings sit left of the text they introduce: section titles at
		// column 0, subsections at 3.
		if ind <= 3 && len(t) < 60 && !strings.HasSuffix(t, ".") {
			for len(indents) > 0 && indents[len(indents)-1] >= ind {
				indents, heads = indents[:len(indents)-1], heads[:len(heads)-1]
			}
			indents, heads = append(indents, ind), append(heads, headingCase(t))
		}
	}
	return refs
}

// headingCase turns "HIGH-LEVEL COMMANDS (PORCELAIN)" into
// "High-level commands (porcelain)"; mixed-case headings are kept.
func headingCase(s string) string {
	if strings.ToUpper(s) != s {
		return s
	}
	s = strings.ToLower(s)
	return strings.ToUpper(s[:1]) + s[1:]
}

var (
	roffFont = regexp.MustCompile(`\\f(\[[^]]*\]|\(..|.)`)
	roffChar = regexp.MustCompile(`\\(\(..|\[[^]]*\]|\*\(..|\*.|&|.)`)
)

// openPage opens a page file, decompressing .gz; it returns nil for
// formats it can't read.
func openPage(path string) (io.Reader, func()) {
	if strings.HasSuffix(path, ".bz2") || strings.HasSuffix(path, ".xz") || strings.HasSuffix(path, ".zst") {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, nil
	}
	if !strings.HasSuffix(path, ".gz") {
		return f, func() { f.Close() }
	}
	zr, err := gzip.NewReader(f)
	if err != nil {
		f.Close()
		return nil, nil
	}
	return zr, func() { zr.Close(); f.Close() }
}

// pageSummary reads the one-line description from a page's NAME section
// ("git-commit \- Record changes to the repository").
func pageSummary(path, name string) string {
	if path == "" {
		return ""
	}
	r, closeFn := openPage(path)
	if r == nil {
		return ""
	}
	defer closeFn()
	sc := bufio.NewScanner(io.LimitReader(r, 32<<10))
	inName := false
	for sc.Scan() {
		l := strings.TrimSpace(sc.Text())
		up := strings.ToUpper(l)
		switch {
		case strings.HasPrefix(up, ".SH"):
			if inName {
				return ""
			}
			inName = strings.Contains(up, "NAME")
		case strings.HasPrefix(l, ".Nd "):
			return cleanRoff(l[4:])
		case inName && l != "" && !strings.HasPrefix(l, "."):
			for _, sep := range []string{`\-`, `\(em`, `\(en`, " - "} {
				if _, d, ok := strings.Cut(l, sep); ok {
					return cleanRoff(d)
				}
			}
			return ""
		}
	}
	return ""
}

// pageSource returns a page's roff source with escapes' backslashes
// removed, so "docker\-container\-ls" reads "docker-container-ls".
func pageSource(path string) string {
	r, closeFn := openPage(path)
	if r == nil {
		return ""
	}
	defer closeFn()
	b, _ := io.ReadAll(io.LimitReader(r, 1<<20))
	return strings.ReplaceAll(string(b), `\`, "")
}

func cleanRoff(s string) string {
	s = roffFont.ReplaceAllString(s, "")
	s = strings.NewReplacer(`\(aq`, "'", `\(dq`, `"`, `\(lq`, `"`, `\(rq`, `"`, `\(cq`, "'", `\(oq`, "'", `\-`, "-", `\e`, `\`).Replace(s)
	s = roffChar.ReplaceAllString(s, "")
	return strings.TrimSuffix(strings.TrimSpace(strings.Trim(s, `"`)), ".")
}

var (
	helpCmdHead  = regexp.MustCompile(`(?i)^(?:\S.*\b)?commands\b.*:\s*$`)
	helpCmdEntry = regexp.MustCompile(`^\s+([a-z][a-z0-9-]*)\*?((?:,\s*[a-z][a-z0-9-]*)*)(?:\s{2,}|\t+)(\S.*)$`)
)

// helpSubcommands reads "Commands:" style lists from --help output, as
// printed by cargo, docker, kubectl, go and most clap- or cobra-based tools.
func helpSubcommands(help string) []Subcommand {
	var subs []Subcommand
	seen := map[string]bool{}
	group := ""
	in := false
	for _, l := range strings.Split(help, "\n") {
		t := strings.TrimSpace(l)
		switch {
		case t == "":
			continue
		case helpCmdHead.MatchString(l):
			in = true
			group = strings.TrimSuffix(t, ":")
			if strings.EqualFold(group, "the commands are") {
				group = "Commands"
			}
			continue
		case l[0] != ' ' && l[0] != '\t':
			in = false
			continue
		}
		if !in {
			continue
		}
		m := helpCmdEntry.FindStringSubmatch(l)
		if m == nil || seen[m[1]] || m[1] == "help" {
			continue
		}
		seen[m[1]] = true
		var aliases []string
		for _, a := range strings.Split(m[2], ",") {
			if a = strings.TrimSpace(a); a != "" {
				aliases = append(aliases, a)
			}
		}
		subs = append(subs, Subcommand{Name: m[1], Aliases: aliases, Desc: strings.TrimSuffix(m[3], "."), Group: group})
	}
	return subs
}

func readSubsCache(key string) ([]Subcommand, bool) {
	dir := cacheDir()
	if dir == "" {
		return nil, false
	}
	b, err := os.ReadFile(filepath.Join(dir, key+".json"))
	if err != nil {
		return nil, false
	}
	var subs []Subcommand
	if json.Unmarshal(b, &subs) != nil {
		return nil, false
	}
	return subs, true
}

func writeSubsCache(key string, subs []Subcommand) { writeCache(key, subs) }
