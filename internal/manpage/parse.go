package manpage

import (
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ansiRe     = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]|\x1b\][^\x07]*\x07`)
	pageMarkRe = regexp.MustCompile(`\(\d[\w]*\)\s*$`)
	spacesRe   = regexp.MustCompile(`\s+`)
	parenRe    = regexp.MustCompile(`\([^()]*\)`)
	shortcutRe = regexp.MustCompile(`^--[\w-]+=[a-z0-9][\w.-]*$`)
	optNameRe  = regexp.MustCompile(`^(?:--?[A-Za-z0-9#@%?][A-Za-z0-9_#@%?+.:-]*|-[,!])$`)
	repeatRe   = regexp.MustCompile(`(?i)(multiple -\S+ options|(?:may|can) be (?:specified|given|used|repeated) (?:multiple|more than once|several|up to)|specif(?:y|ied) (?:it )?(?:multiple|more than once|several) times|repeated|each additional|more -\S+ options)`)
)

// skipSections never contain options worth presenting.
var skipSections = map[string]bool{
	"NAME": true, "SYNOPSIS": true, "SEE ALSO": true, "ENVIRONMENT": true,
	"EXIT STATUS": true, "FILES": true, "HISTORY": true, "AUTHORS": true,
	"AUTHOR": true, "BUGS": true, "STANDARDS": true, "COPYRIGHT": true,
	"REPORTING BUGS": true, "EXAMPLES": true, "EXAMPLE": true,
	"DIAGNOSTICS": true, "COMPATIBILITY": true, "CAVEATS": true,
	"RETURN VALUES": true, "EXIT CODES": true, "SECURITY CONSIDERATIONS": true,
	"LEGACY DESCRIPTION": true, "IMPLEMENTATION NOTES": true,
}

// Clean turns rendered man output into plain text: it removes backspace
// overstriking (bold/underline), ANSI escape sequences and tabs.
func Clean(raw string) string {
	raw = ansiRe.ReplaceAllString(raw, "")
	var b strings.Builder
	b.Grow(len(raw))
	runes := make([]rune, 0, len(raw))
	for _, r := range raw {
		if r == '\b' {
			if len(runes) > 0 {
				runes = runes[:len(runes)-1]
			}
			continue
		}
		runes = append(runes, r)
	}
	col := 0
	for _, r := range runes {
		switch r {
		case '\t':
			n := 8 - col%8
			b.WriteString(strings.Repeat(" ", n))
			col += n
		case '\n':
			b.WriteRune(r)
			col = 0
		case '\r', '​':
		case ' ':
			b.WriteByte(' ')
			col++
		default:
			b.WriteRune(r)
			col++
		}
	}
	lines := strings.Split(b.String(), "\n")
	for i, l := range lines {
		lines[i] = strings.TrimRight(l, " ")
	}
	return strings.TrimSpace(strings.Join(lines, "\n")) + "\n"
}

type line struct {
	ind  int    // indentation in columns
	s    string // trimmed text
	raw  string // full line
	sect string // top-level section
	sub  string // subsection
}

func (l line) blank() bool { return l.s == "" }

func indentOf(s string) int {
	n := 0
	for _, r := range s {
		if r != ' ' {
			break
		}
		n++
	}
	return n
}

// Parse parses cleaned manual text (see Clean). source is "man" or "help".
func Parse(command, text, source string) *Spec {
	spec := &Spec{Command: command, Manual: text, Source: source}
	raw := strings.Split(strings.TrimRight(text, "\n"), "\n")

	// Drop the page header and footer ("LS(1)  User Commands  LS(1)").
	if source == "man" {
		first, last := -1, -1
		for i, l := range raw {
			if strings.TrimSpace(l) != "" {
				if first < 0 {
					first = i
				}
				last = i
			}
		}
		if last >= 0 && last != first && pageMarkRe.MatchString(raw[last]) && indentOf(raw[last]) == 0 {
			raw = raw[:last]
		}
		if first >= 0 && pageMarkRe.MatchString(raw[first]) && indentOf(raw[first]) == 0 {
			raw = raw[first+1:]
		}
	}

	lines := make([]line, 0, len(raw))
	sect, sub := "", ""
	if source == "help" {
		sect = "OPTIONS"
	}
	for _, r := range raw {
		l := line{ind: indentOf(r), s: strings.TrimSpace(r), raw: r}
		if !l.blank() {
			switch {
			case source == "man" && l.ind == 0:
				sect, sub = strings.ToUpper(l.s), ""
				l.sect = sect
				lines = append(lines, line{sect: sect, s: "", raw: ""}) // heading marker
				lines[len(lines)-1].ind = -1
				continue
			case source == "man" && l.ind > 0 && l.ind <= 4 && !strings.HasPrefix(l.s, "-") && utf8.RuneCountInString(l.s) < 60:
				sub = strings.TrimSuffix(l.s, ":")
				lines = append(lines, line{ind: -2, sect: sect, sub: sub})
				continue
			case source == "help" && l.ind == 0 && strings.HasSuffix(l.s, ":") && !strings.HasPrefix(l.s, "-"):
				sub = strings.TrimSuffix(l.s, ":")
				lines = append(lines, line{ind: -2, sect: sect, sub: sub})
				continue
			}
		}
		l.sect, l.sub = sect, sub
		lines = append(lines, l)
	}

	spec.Summary = summaryFrom(lines, source)
	spec.Synopsis = synopsisFrom(lines, source)
	spec.Args = parseArgs(spec.Synopsis, command)
	spec.Examples = findExamples(lines, command, source)

	p := &parser{lines: lines, spec: spec}
	p.findOptions()
	p.configOptions()
	spec.Description = p.description()
	p.detectGroups()
	p.finishOptions()
	return spec
}

type parser struct {
	lines []line
	spec  *Spec
	// option start lines, used for description and group detection.
	optLines []int
	optEnd   []int
	paras    []para         // paragraphs outside option descriptions
	blocks   map[int][]line // each option's own lines, by option index
}

type para struct {
	text string
	sect string
	sub  string
	line int
}

func sectionBody(lines []line, name string) []line {
	var out []line
	in := false
	for _, l := range lines {
		if l.ind == -1 {
			in = l.sect == name
			continue
		}
		if in && l.ind >= 0 {
			out = append(out, l)
		}
	}
	return out
}

func summaryFrom(lines []line, source string) string {
	if source == "help" {
		return ""
	}
	var parts []string
	in := false
	for _, l := range lines {
		switch {
		case l.ind == -1:
			in = l.sect == "NAME"
		case !in:
		case l.ind == -2:
			in = false // npm puts the whole page in subsections of NAME
		case !l.blank():
			parts = append(parts, l.s)
		}
	}
	s := strings.Join(parts, " ")
	for _, sep := range []string{" – ", " — ", " - ", " -- ", " \\- "} {
		if i := strings.Index(s, sep); i >= 0 {
			return strings.TrimSpace(s[i+len(sep):])
		}
	}
	return s
}

func synopsisFrom(lines []line, source string) string {
	if source == "help" {
		var out []string
		collecting := false
		for _, l := range lines {
			if l.ind < 0 {
				continue
			}
			low := strings.ToLower(l.s)
			if strings.HasPrefix(low, "usage:") || (collecting && strings.HasPrefix(low, "or:")) {
				out = append(out, l.s)
				collecting = true
				continue
			}
			if collecting {
				if l.blank() || l.ind == 0 {
					break
				}
				out[len(out)-1] += " " + l.s
			}
		}
		return strings.Join(out, "\n")
	}
	body := sectionBody(lines, "SYNOPSIS")
	base := 1 << 30
	for _, l := range body {
		if !l.blank() && l.ind < base {
			base = l.ind
		}
	}
	var out []string
	for _, l := range body {
		if l.blank() {
			continue
		}
		if l.ind <= base || len(out) == 0 {
			out = append(out, l.s)
		} else {
			out[len(out)-1] += " " + l.s
		}
	}
	for i := range out {
		out[i] = spacesRe.ReplaceAllString(out[i], " ")
	}
	return strings.Join(out, "\n")
}

// tagParts splits "-a, --all" into ["-a", "--all"] but keeps commas that
// belong to arguments ("-o key=val[,key=val]").
func tagParts(tag string) []string {
	var parts []string
	depth := 0
	start := 0
	paired, commas := false, false
	for i := 0; i < len(tag); i++ {
		switch tag[i] {
		case '[', '<', '{', '(':
			depth++
		case ']', '>', '}', ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				rest := strings.TrimLeft(tag[i+1:], " ")
				rest = strings.TrimPrefix(rest, "or ") // "-a x, -b x, or -c x"
				if strings.HasPrefix(rest, "-") || strings.HasPrefix(rest, "+") {
					parts = append(parts, strings.TrimSpace(tag[start:i]))
					start = i + 1
					commas = true
				}
			}
		case ' ':
			if i < start {
				continue // inside " or "
			}
			cur := strings.TrimSpace(tag[start:i])
			switch {
			case depth > 0 || cur == "":
			case strings.HasPrefix(tag[i:], " or -"):
				// "-b addr or --bind-address addr" (JDK pages)
				parts = append(parts, cur)
				start = i + 4
			case strings.HasPrefix(tag[i:], " --") && optNameRe.MatchString(cur) && !strings.HasPrefix(cur, "--"):
				if shortcutRe.MatchString(strings.TrimSpace(tag[i:])) {
					// "-c --order=cpu": a flag standing for that setting.
					return []string{cur}
				}
				// "-a --all", "-D --directory=PATH" (systemd, binutils)
				parts = append(parts, cur)
				start = i + 1
				paired = true
			}
		case '|':
			// "--lint|--enable-checks"
			if depth == 0 && strings.HasPrefix(tag[i+1:], "-") {
				parts = append(parts, strings.TrimSpace(tag[start:i]))
				start = i + 1
			}
		}
	}
	if paired && commas {
		// "-1 --base, -2 --ours, -3 --theirs": several options, one entry.
		return []string{""}
	}
	for k := range parts {
		parts[k] = strings.TrimSpace(strings.TrimPrefix(parts[k], "or "))
	}
	last := strings.TrimSpace(tag[start:])
	return append(parts, strings.TrimSpace(strings.TrimPrefix(last, "or ")))
}

type tagInfo struct {
	repeat   bool // "--verbose..." (clap)
	names    []string
	arg      string
	optional bool
	longEq   bool
	argOf    string // the name the argument was given with
}

// parseTag parses an option tag such as "-w, --width=COLS",
// "--color[=WHEN]", "-D format" or "-d, --data <data>".
func parseTag(tag string) (tagInfo, bool) {
	var ti tagInfo
	tag = strings.TrimSpace(tag)
	if !strings.HasPrefix(tag, "-") || len(tag) < 2 || utf8.RuneCountInString(tag) > 80 {
		return ti, false
	}
	parts := tagParts(tag)
	shortcut := len(parts) == 1 && parts[0] != tag // "-EB --endian=big"
	for _, part := range parts {
		if part == "" {
			return ti, false
		}
		name, arg := part, ""
		optional, eq := false, false
		if i := strings.IndexAny(part, " =[<"); i > 0 {
			name, arg = part[:i], part[i:]
			// "-,": a lone comma option, as in BSD ls.
		}
		if name == "-" && strings.HasPrefix(part, "-,") {
			name, arg = "-,", strings.TrimPrefix(part, "-,")
		}
		if strings.HasSuffix(name, "...") && len(name) > 4 {
			name = strings.TrimSuffix(name, "...")
			ti.repeat = true
		}
		if !optNameRe.MatchString(name) || strings.HasSuffix(name, ".") || strings.HasSuffix(name, ":") || name == "--" {
			return ti, false
		}
		arg = strings.TrimSpace(arg)
		switch {
		case strings.HasPrefix(arg, "[="):
			optional, eq = true, true
			arg = strings.TrimSuffix(strings.TrimPrefix(arg, "[="), "]")
		case strings.HasPrefix(arg, "=["):
			optional, eq = true, true
			arg = strings.TrimSuffix(strings.TrimPrefix(arg, "=["), "]")
		case strings.HasPrefix(arg, "="):
			eq = true
			arg = arg[1:]
		case strings.HasPrefix(arg, "[") && strings.HasSuffix(arg, "]") && !strings.Contains(arg[1:len(arg)-1], "]"):
			optional = true
			arg = arg[1 : len(arg)-1]
		}
		arg = strings.TrimSpace(arg)
		arg = trimEllipsis(arg)
		if strings.HasPrefix(arg, "<") && strings.HasSuffix(arg, ">") {
			arg = trimEllipsis(arg[1 : len(arg)-1])
		}
		if !strings.HasPrefix(name, "--") && len(name) > 2 && isUpperWord(name[1:]) && arg == "" && !shortcut {
			continue // "-NUM" style pseudo options
		}
		if arg != "" && !plausibleArg(arg) {
			return ti, false
		}
		ti.names = append(ti.names, name)
		if arg != "" && ti.arg == "" {
			ti.arg, ti.argOf = arg, name
		}
		if optional {
			ti.optional = true
		}
		if eq && strings.HasPrefix(name, "--") {
			ti.longEq = true
		}
	}
	return ti, len(ti.names) > 0
}

// splitEntry reads a tag naming several options, each a short and a long
// name ("-1 --base, -2 --ours, -3 --theirs, -0"), or returns nil.
func splitEntry(tag string) []tagInfo {
	groups := strings.Split(tag, ", ")
	if len(groups) < 2 {
		return nil
	}
	var out []tagInfo
	paired := false
	for _, g := range groups {
		ti, ok := parseTag(g)
		if !ok || len(ti.names) > 2 {
			return nil
		}
		paired = paired || len(ti.names) == 2
		out = append(out, ti)
	}
	if !paired {
		return nil
	}
	return out
}

// trimEllipsis drops a trailing "..." from an argument ("dir ...",
// "<ID,...>": the option takes one or more of them), and the separator
// before it.
func trimEllipsis(arg string) string {
	if a := strings.TrimSpace(strings.TrimSuffix(arg, "...")); a != arg && a != "" {
		return strings.TrimSpace(strings.TrimRight(a, ",:;"))
	}
	return arg
}

var (
	configNameRe = regexp.MustCompile(`^[a-z][a-z0-9]*(?:-[a-z0-9]+)*$`)
	configTypeRe = regexp.MustCompile(`^(?:o\s+|•\s*)?Type:\s*(.*)$`)
	quotedRe     = regexp.MustCompile(`"([^"]+)"`)
)

// configOptions reads settings documented as a subsection each, named
// without dashes and followed by their default and type, as npm's
// manuals do:
//
//	save
//	    o   Default: true
//	    o   Type: Boolean
//	    Save installed packages to a package.json file as dependencies.
//
// Each is the option --save. "Type: Boolean" makes a flag; quoted values
// ("dev", "optional", or "peer") a dropdown; anything else a text field.
func (p *parser) configOptions() {
	ls := p.lines
	seen := map[string]bool{}
	for _, o := range p.spec.Options {
		for _, n := range o.Names {
			seen[n] = true
		}
	}
	for i, l := range ls {
		if l.ind != -2 || !configNameRe.MatchString(l.sub) || seen["--"+l.sub] {
			continue
		}
		end := i + 1
		for end < len(ls) && ls[end].ind >= 0 {
			end++
		}
		body := ls[i+1 : end]
		typ, hasDefault := "", false
		for _, b := range body[:min(len(body), 8)] {
			if m := configTypeRe.FindStringSubmatch(b.s); m != nil {
				typ = m[1]
			}
			if strings.Contains(b.s, "Default:") {
				hasDefault = true
			}
		}
		if typ == "" || !hasDefault {
			continue
		}
		o := Option{Names: []string{"--" + l.sub}, LongEquals: true, Section: "Configuration"}
		switch t := strings.ToLower(typ); {
		case strings.Contains(t, "boolean") && !strings.Contains(t, "string"):
			// a flag (--save; --no-save is npm's way to turn it off)
		case strings.Contains(t, "number"):
			o.Arg = "NUMBER"
		default:
			o.Arg = "VALUE"
			for _, m := range quotedRe.FindAllStringSubmatch(typ, -1) {
				o.Choices = append(o.Choices, m[1])
			}
			if len(o.Choices) < 2 {
				o.Choices = nil
			}
		}
		if strings.Contains(typ, "can be set multiple times") {
			o.Repeatable = true
		}
		// The description, without the Default:, Type: and DEPRECATED
		// bullets, which may wrap onto further lines.
		var text []line
		inBullet := false
		for _, b := range body {
			switch {
			case b.blank():
				inBullet = false
			case strings.HasPrefix(b.s, "o ") && (strings.Contains(b.s, "Default:") || strings.Contains(b.s, "Type:") || strings.Contains(b.s, "DEPRECATED")):
				inBullet = true
			}
			if !inBullet {
				text = append(text, b)
			}
		}
		o.Desc = joinParagraphs("", text)
		if p.blocks == nil {
			p.blocks = map[int][]line{}
		}
		p.blocks[len(p.spec.Options)] = body
		p.optLines = append(p.optLines, i)
		p.optEnd = append(p.optEnd, end)
		seen["--"+l.sub] = true
		p.spec.Options = append(p.spec.Options, o)
	}
}

// mergeTags joins the names of stacked tag lines into one option.
func mergeTags(a, b tagInfo) tagInfo {
	for _, n := range b.names {
		if indexOfString(a.names, n) < 0 {
			a.names = append(a.names, n)
		}
	}
	if a.arg == "" {
		a.arg, a.argOf = b.arg, b.argOf
	}
	a.optional = a.optional || b.optional
	a.longEq = a.longEq || b.longEq
	a.repeat = a.repeat || b.repeat
	return a
}

func isUpperWord(s string) bool {
	for _, r := range s {
		if !unicode.IsUpper(r) && r != '_' {
			return false
		}
	}
	return s != ""
}

// plausibleArg rejects "arguments" that are really prose, which happens when
// body text that starts with an option name is mistaken for a tag.
func plausibleArg(arg string) bool {
	words := strings.Fields(arg)
	if len(words) > 4 {
		return false
	}
	if len(words) > 1 {
		for _, w := range words[1:] {
			switch strings.ToLower(strings.Trim(w, ".,;:")) {
			case "the", "is", "are", "to", "and", "or", "option", "options", "of", "that", "this", "with", "for", "if":
				if !strings.ContainsAny(arg, "<>[]") {
					return false
				}
			}
		}
	}
	first := []rune(words[0])
	if unicode.IsUpper(first[0]) && len(first) > 1 && unicode.IsLower(first[1]) && len(words) > 1 {
		return false // "That is, you ..."
	}
	return !strings.HasSuffix(arg, ".")
}

// tagEnd is where a tag ends and its same-line description starts: at the
// first run of spaces, except one after a comma that leads to the next name
// ("-T,  --timeout=SECONDS   set all timeouts", wget).
func tagEnd(s string) int {
	from := 0
	for {
		i := strings.Index(s[from:], "  ")
		if i < 0 {
			return -1
		}
		i += from
		if i > 0 && s[i-1] == ',' && strings.HasPrefix(strings.TrimLeft(s[i:], " "), "-") {
			from = i + 2
			continue
		}
		return i
	}
}

// splitInline splits a tag line into tag and same-line description.
func splitInline(l line, delta int) (tag, rest string, col int) {
	rs := []rune(l.raw)
	c := l.ind + delta
	exact := delta > 0 && len(rs) > c && rs[c-1] == ' ' && rs[c] != ' '
	if exact && rs[c-2] == ' ' {
		tag, rest := strings.TrimSpace(string(rs[:c])), strings.TrimSpace(string(rs[c:]))
		if !strings.HasSuffix(tag, ",") || !strings.HasPrefix(rest, "-") { // not "-T,  --timeout"
			return tag, rest, c
		}
	}
	if i := tagEnd(l.s); i > 0 {
		if _, ok := parseTag(l.s[:i]); ok {
			r := strings.TrimLeft(l.s[i:], " ")
			return l.s[:i], r, l.ind + utf8.RuneCountInString(l.s[:len(l.s)-len(r)])
		}
	}
	if exact {
		// A tag that exactly fills the indent: "--zero end each output line".
		if t := strings.TrimSpace(string(rs[:c])); !strings.Contains(t, " ") {
			return t, strings.TrimSpace(string(rs[c:])), c
		}
	}
	return l.s, "", 0
}

type cand struct {
	idx  int
	last int // the last of its tag lines: names can be stacked one per line
	tag  tagInfo
	rest string
	col  int
}

func (p *parser) nextNonBlank(i int) int {
	for j := i + 1; j < len(p.lines); j++ {
		if p.lines[j].ind < 0 {
			return -1
		}
		if !p.lines[j].blank() {
			return j
		}
	}
	return -1
}

func (p *parser) findOptions() {
	ls := p.lines
	// Learn how far descriptions are indented relative to their tags.
	deltas := map[int]int{}
	for i, l := range ls {
		if l.ind <= 0 || !strings.HasPrefix(l.s, "-") || skipSections[l.sect] {
			continue
		}
		if _, ok := parseTag(l.s); !ok {
			continue
		}
		if j := p.nextNonBlank(i); j > 0 && ls[j].ind > l.ind {
			deltas[ls[j].ind-l.ind]++
		}
	}
	delta := mode(deltas, 0)

	var cands []cand
	indents := map[string]map[int]int{}
	skipTo := -1
	for i, l := range ls {
		if i <= skipTo || l.ind < 0 || l.blank() || !strings.HasPrefix(l.s, "-") || skipSections[l.sect] {
			continue
		}
		last := i
		var stacked tagInfo
		if ti, ok := parseTag(l.s); ok {
			// "-a" over "--echo-all" over the description (DocBook pages:
			// PostgreSQL, git): one option, named on each line.
			stacked = ti
			for k := i + 1; k < len(ls) && k-i < 6; k++ {
				n := ls[k]
				if n.ind != l.ind || !strings.HasPrefix(n.s, "-") || strings.Contains(n.s, "  ") {
					break
				}
				nt, ok := parseTag(n.s)
				if !ok {
					break
				}
				stacked = mergeTags(stacked, nt)
				last = k
			}
		}
		j := p.nextNonBlank(last)
		if last > i {
			if j < 0 || ls[j].ind <= l.ind {
				continue
			}
			skipTo = last
			cands = append(cands, cand{idx: i, last: last, tag: stacked})
			if indents[l.sect] == nil {
				indents[l.sect] = map[int]int{}
			}
			indents[l.sect][l.ind]++
			continue
		}
		tag, rest, col := splitInline(l, delta)
		if j > 0 && ls[j].ind > l.ind && rest != "" && !strings.Contains(l.s, "  ") {
			// "-Bnewer file" with the description on the following lines.
			if _, ok := parseTag(l.s); ok {
				tag, rest, col = l.s, "", 0
			}
		}
		ti, ok := parseTag(tag)
		tis := []tagInfo{ti}
		if !ok {
			// "-1 --base, -2 --ours, -3 --theirs": several options sharing
			// one description.
			if tis = splitEntry(tag); tis == nil {
				continue
			}
		}
		if rest == "" {
			if j < 0 || ls[j].ind <= l.ind {
				continue
			}
		} else if j == i+1 && ls[j].ind <= l.ind {
			// Same-line description must be followed by a break, its own
			// continuation, or another tag; otherwise it is body text.
			nt, _, _ := splitInline(ls[j], delta)
			if _, ok := parseTag(nt); !ok || !strings.HasPrefix(ls[j].s, "-") {
				continue
			}
		}
		for _, ti := range tis {
			cands = append(cands, cand{idx: i, last: i, tag: ti, rest: rest, col: col})
		}
		if indents[l.sect] == nil {
			indents[l.sect] = map[int]int{}
		}
		indents[l.sect][l.ind]++
	}
	// Keep candidates at the dominant tag indentation of their section;
	// deeper ones are examples or nested lists inside descriptions.
	limit := map[string]int{}
	for s, m := range indents {
		limit[s] = mode(m, 0) + 4
	}
	kept := cands[:0]
	for _, c := range cands {
		lim := limit[ls[c.idx].sect]
		if strings.HasPrefix(ls[c.idx].s, "--") {
			lim += 4 // long-only names lined up under "-x, --long" (wget)
		}
		if ls[c.idx].ind <= lim {
			kept = append(kept, c)
		}
	}
	cands = kept

	seen := map[string]bool{}
	optOf := map[string]int{} // option index by name
	isStart := map[int]bool{}
	for _, c := range cands {
		isStart[c.idx] = true
	}
	for _, c := range cands {
		l := ls[c.idx]
		// Description block: until the next tag, or a non-blank line that is
		// not indented deeper than the tag.
		end := c.last + 1
		for end < len(ls) {
			n := ls[end]
			if n.ind < 0 || isStart[end] || (!n.blank() && n.ind <= l.ind) {
				break
			}
			end++
		}
		p.optLines = append(p.optLines, c.idx)
		p.optEnd = append(p.optEnd, end)

		// An option documented twice keeps its first entry; names already
		// taken ("-e --echo" before "--echo=MODE") leave the rest.
		var fresh []string
		for _, n := range c.tag.names {
			if !seen[n] {
				fresh = append(fresh, n)
			}
		}
		if len(fresh) == 0 {
			continue
		}
		if len(fresh) < len(c.tag.names) {
			k := -1
			for _, n := range c.tag.names {
				if seen[n] {
					k = optOf[n]
					break
				}
			}
			o := &p.spec.Options[k]
			switch {
			case c.tag.arg == "" || strings.EqualFold(o.Arg, c.tag.arg):
				// More names for an option already read ("--existing,
				// --ignore-non-existing" after "--existing").
				o.Names = append(o.Names, fresh...)
				for _, n := range fresh {
					seen[n], optOf[n] = true, k
				}
				continue
			case c.tag.argOf != "" && indexOfString(fresh, c.tag.argOf) < 0:
				// "-p, --indicator-style=slash" after --indicator-style=WORD:
				// -p is a flag for that one value.
				c.tag.arg, c.tag.optional, c.tag.longEq = "", false, false
			}
		}
		c.tag.names = fresh
		for _, n := range c.tag.names {
			seen[n], optOf[n] = true, len(p.spec.Options)
		}
		desc := joinParagraphs(c.rest, ls[c.last+1:end])
		if p.blocks == nil {
			p.blocks = map[int][]line{}
		}
		p.blocks[len(p.spec.Options)] = ls[c.last+1 : end]
		section := l.sub
		if section == "" {
			section = titleCase(l.sect)
		}
		p.spec.Options = append(p.spec.Options, Option{
			Names:       c.tag.names,
			Arg:         c.tag.arg,
			ArgOptional: c.tag.optional,
			LongEquals:  c.tag.longEq,
			Desc:        desc,
			Section:     section,
			Repeatable:  c.tag.repeat,
		})
	}

	// Paragraphs outside option descriptions.
	inOpt := make([]bool, len(ls))
	for k, s := range p.optLines {
		for i := s; i < p.optEnd[k]; i++ {
			inOpt[i] = true
		}
	}
	var cur []string
	start := -1
	flush := func(i int) {
		if len(cur) > 0 {
			p.paras = append(p.paras, para{text: joinWrapped(cur), sect: ls[start].sect, sub: ls[start].sub, line: start})
		}
		cur, start = nil, -1
	}
	for i, l := range ls {
		if inOpt[i] || l.ind < 0 || l.blank() {
			flush(i)
			continue
		}
		if start < 0 {
			start = i
		}
		cur = append(cur, l.s)
	}
	flush(len(ls))
}

func mode(m map[int]int, def int) int {
	best, bestN := def, 0
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	for _, k := range keys {
		if m[k] > bestN {
			best, bestN = k, m[k]
		}
	}
	return best
}

// joinWrapped joins wrapped lines, undoing hyphenation.
func joinWrapped(parts []string) string {
	var b strings.Builder
	for i, s := range parts {
		if i > 0 {
			prev := b.String()
			switch {
			case strings.HasSuffix(prev, "‐"):
				t := strings.TrimSuffix(prev, "‐")
				b.Reset()
				b.WriteString(t)
			case strings.HasSuffix(prev, "-") && len(prev) > 1 && isLetter(prev[len(prev)-2]) && s != "" && isLower(s[0]):
			default:
				b.WriteByte(' ')
			}
		}
		b.WriteString(s)
	}
	return strings.TrimSpace(spacesRe.ReplaceAllString(b.String(), " "))
}

func isLetter(c byte) bool { return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') }
func isLower(c byte) bool  { return c >= 'a' && c <= 'z' }

// joinParagraphs builds a description from block lines. Lines indented
// deeper than the block's base (examples) are kept verbatim.
func joinParagraphs(first string, block []line) string {
	base := 1 << 30
	for _, l := range block {
		if !l.blank() && l.ind >= 0 && l.ind < base {
			base = l.ind
		}
	}
	var paras []string
	var cur []string
	var pre []string
	flush := func() {
		if len(cur) > 0 {
			paras = append(paras, joinWrapped(cur))
			cur = nil
		}
		if len(pre) > 0 {
			paras = append(paras, strings.Join(pre, "\n"))
			pre = nil
		}
	}
	if first != "" {
		cur = append(cur, first)
	}
	for _, l := range block {
		if l.ind < 0 {
			continue
		}
		if l.blank() {
			flush()
			continue
		}
		if l.ind > base+1 && (len(cur) == 0 || l.ind >= base+4) {
			if len(cur) > 0 {
				paras = append(paras, joinWrapped(cur))
				cur = nil
			}
			pre = append(pre, "  "+strings.TrimRight(l.raw[min(len(l.raw), base):], " "))
			continue
		}
		if len(pre) > 0 {
			paras = append(paras, strings.Join(pre, "\n"))
			pre = nil
		}
		cur = append(cur, l.s)
	}
	flush()
	return strings.Join(paras, "\n\n")
}

func titleCase(s string) string {
	words := strings.Fields(strings.ToLower(s))
	for i, w := range words {
		r, n := utf8.DecodeRuneInString(w)
		words[i] = string(unicode.ToUpper(r)) + w[n:]
	}
	return strings.Join(words, " ")
}

// description returns the introductory paragraphs of DESCRIPTION.
func (p *parser) description() string {
	firstOpt := 1 << 30
	if len(p.optLines) > 0 {
		firstOpt = p.optLines[0]
	}
	var out []string
	for _, pa := range p.paras {
		want := pa.sect == "DESCRIPTION"
		if p.spec.Source == "help" {
			low := strings.ToLower(pa.text)
			want = pa.line < firstOpt && !strings.HasPrefix(low, "usage:") && !strings.HasPrefix(low, "or:")
		}
		if !want {
			continue
		}
		if pa.line > firstOpt && len(out) > 0 {
			break
		}
		out = append(out, pa.text)
		if len(out) >= 4 {
			break
		}
	}
	return strings.Join(out, "\n\n")
}

var (
	exclusiveRe = regexp.MustCompile(`(?i)(override each other|override one another|mutually exclusive|cancel each other|last one specified|last one given)`)
	exactlyOne  = regexp.MustCompile(`(?i)\b(exactly one of|only one of (these|them|the following)|one of the following (options )?must)\b`)
	nameTokRe   = regexp.MustCompile(`(?:^|[\s(“"'‘/])(--?[A-Za-z0-9@%#][\w-]*)`)
)

func (p *parser) indexByName() map[string]int {
	m := map[string]int{}
	for i, o := range p.spec.Options {
		for _, n := range o.Names {
			m[n] = i
		}
	}
	return m
}

// detectGroups finds sets of options documented as mutually exclusive.
func (p *parser) detectGroups() {
	byName := p.indexByName()
	used := map[int]bool{}
	add := func(label string, members []int) {
		var ms []int
		for _, m := range members {
			if !used[m] && !p.spec.Options[m].TakesArg() {
				ms = append(ms, m)
			}
		}
		if len(ms) < 2 {
			return
		}
		sort.Ints(ms)
		for _, m := range ms {
			used[m] = true
		}
		p.spec.Groups = append(p.spec.Groups, Group{Label: label, Members: ms})
	}

	// "Exactly one of them must be given" in a subsection preamble.
	for _, pa := range p.paras {
		if pa.sub == "" || !exactlyOne.MatchString(pa.text) {
			continue
		}
		var ms []int
		for i, o := range p.spec.Options {
			if o.Section == pa.sub {
				ms = append(ms, i)
			}
		}
		add(pa.sub, ms)
	}

	// "The -1, -C, -x, and -l options all override each other; the last one
	// specified determines the format used." in general text.
	for _, pa := range p.paras {
		if !exclusiveRe.MatchString(pa.text) {
			continue
		}
		for _, sent := range strings.SplitAfter(pa.text, ". ") {
			if !exclusiveRe.MatchString(sent) {
				continue
			}
			sent = strings.TrimSuffix(strings.TrimSpace(sent), ".")
			label := "Choose one"
			if m := determinesRe.FindStringSubmatch(sent); m != nil {
				label = capitalize(strings.TrimSpace(m[1]))
			}
			add(label, p.namesIn(sent, byName))
		}
	}

	// "This option is mutually exclusive to -f, --fail." / "This option
	// cancels the -P option." inside an option's own description.
	opts := p.spec.Options
	for i := range opts {
		for _, sent := range sentences(opts[i].Desc) {
			if !conflictRe.MatchString(sent) {
				continue
			}
			for _, j := range p.namesIn(sent, byName) {
				if j != i {
					opts[i].Conflicts = appendUnique(opts[i].Conflicts, opts[j].Names[0])
					opts[j].Conflicts = appendUnique(opts[j].Conflicts, opts[i].Names[0])
				}
			}
		}
	}
}

var (
	determinesRe = regexp.MustCompile(`(?i)determines the (.+?)(?: used)?\s*$`)
	conflictRe   = regexp.MustCompile(`(?i)(mutually exclusive|cancels the|cancels any|overrides the|override each other)`)
)

func (p *parser) namesIn(sent string, byName map[string]int) []int {
	var ms []int
	seen := map[int]bool{}
	for _, m := range nameTokRe.FindAllStringSubmatch(sent, -1) {
		if i, ok := byName[m[1]]; ok && !seen[i] {
			seen[i] = true
			ms = append(ms, i)
		}
	}
	return ms
}

func appendUnique(list []string, s string) []string {
	for _, x := range list {
		if x == s {
			return list
		}
	}
	return append(list, s)
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	r, n := utf8.DecodeRuneInString(s)
	return string(unicode.ToUpper(r)) + s[n:]
}

func sentences(t string) []string {
	var out []string
	start := 0
	for i := 0; i < len(t); i++ {
		if (t[i] == '.' || t[i] == ';') && (i+1 == len(t) || t[i+1] == ' ' || t[i+1] == '\n') && i > 0 && t[i-1] != ' ' {
			// avoid splitting "e.g." and "etc."
			if i >= 3 && (t[i-3:i] == "e.g" || t[i-3:i] == "i.e" || t[i-3:i] == "etc") {
				continue
			}
			out = append(out, strings.TrimSpace(t[start:i+1]))
			start = i + 1
		}
	}
	if s := strings.TrimSpace(t[start:]); s != "" {
		out = append(out, s)
	}
	return out
}

func (p *parser) finishOptions() {
	opts := p.spec.Options
	for i := range opts {
		o := &opts[i]
		o.Label = makeLabel(o)
		if !o.TakesArg() {
			o.Kind = KindFlag
			o.Repeatable = o.Repeatable || repeatRe.MatchString(firstParas(o.Desc, 2))
			continue
		}
		o.Kind = kindForArg(o.Arg, o.Desc)
		if o.Kind == KindString && len(o.Choices) >= 2 {
			o.Kind = KindChoice // given with the option (npm's "Type:")
		} else if o.Kind == KindString {
			if ch := p.choicesFor(o); len(ch) >= 2 {
				o.Kind = KindChoice
				o.Choices = ch
			}
		}
		o.Label = refineLabel(o)
		o.Notes = p.notesFor(o)
	}
	byName := p.indexByName()
	for i := range opts {
		if opts[i].TakesArg() {
			p.valueDescs(&opts[i], p.blocks[i], byName)
		}
		opts[i].Examples = optionExamples(p.blocks[i], p.spec.Command)
	}
}

// notesFor collects general paragraphs that discuss the option or its
// argument placeholder, e.g. GNU ls's "The WHEN argument defaults to ...".
func (p *parser) notesFor(o *Option) []string {
	var pats []*regexp.Regexp
	if isUpperWord(o.Arg) && len(o.Arg) >= 3 && !genericArg[o.Arg] {
		pats = append(pats, regexp.MustCompile(`\b(?:The|the) `+regexp.QuoteMeta(o.Arg)+` argument\b`))
	}
	for _, l := range o.Long() {
		pats = append(pats, regexp.MustCompile(regexp.QuoteMeta(l)+`\b`))
	}
	var out []string
	for _, pa := range p.paras {
		if skipSections[pa.sect] || pa.sect == "DESCRIPTION" && strings.Contains(p.spec.Description, pa.text) {
			continue
		}
		for _, re := range pats {
			if re.MatchString(pa.text) {
				out = append(out, pa.text)
				break
			}
		}
		if len(out) == 3 {
			break
		}
	}
	return out
}

func firstParas(s string, n int) string {
	parts := strings.SplitN(s, "\n\n", n+1)
	if len(parts) > n {
		parts = parts[:n]
	}
	return strings.Join(parts, " ")
}

var (
	numberArgRe = regexp.MustCompile(`(?i)^(n|num|number|count|cols|columns|lines|width|height|depth|maxdepth|mindepth|levels?|seconds|secs?|ms|milliseconds|timeout|port|retries|times|max|limit|fractional seconds|integer|int)$`)
	pathArgRe   = regexp.MustCompile(`(?i)(file|dir|path|folder|archive|directory)`)
)

func kindForArg(arg, desc string) Kind {
	a := strings.Trim(strings.ToLower(arg), "<>[]{}.… ")
	switch {
	case strings.ContainsAny(arg, "|{"):
		return KindString // choices handled later
	case numberArgRe.MatchString(a), strings.HasSuffix(a, "num"), strings.HasSuffix(a, "count"), strings.HasSuffix(a, "seconds"):
		return KindNumber
	case pathArgRe.MatchString(a) && !strings.Contains(a, "type") && !strings.Contains(a, "name ") && !strings.Contains(a, "format"):
		return KindPath
	}
	return KindString
}

// refineLabel improves labels of options that take values: a description
// that merely lists the values ("across -x, commas -m, ...") is replaced by
// the option's name, and placeholders read as words ("by SIZE" → "by size").
func refineLabel(o *Option) string {
	l := o.Label
	hits := 0
	for _, c := range o.Choices {
		if strings.Contains(strings.ToLower(l), strings.ToLower(c)) {
			hits++
		}
	}
	if hits >= 2 || hits == len(o.Choices) && hits > 0 && len(l) < 3*len(o.Choices[0]) {
		if long := o.Long(); len(long) > 0 {
			return capitalize(strings.ReplaceAll(strings.TrimLeft(long[0], "-"), "-", " "))
		}
	}
	if isUpperWord(o.Arg) && len(o.Arg) > 1 {
		l = regexp.MustCompile(`\b`+regexp.QuoteMeta(o.Arg)+`\b`).ReplaceAllString(l, strings.ToLower(o.Arg))
		l = capitalize(l)
	}
	return l
}

var bracketNoteRe = regexp.MustCompile(`\s*\[(?:possible values|default|env|aliases?|values?)[^\]]*\]`)

func makeLabel(o *Option) string {
	s := firstParas(o.Desc, 1)
	s = strings.TrimSpace(s)
	// Drop a leading parenthetical: "(The lowercase letter “ell”.) List ..."
	if strings.HasPrefix(s, "(") {
		if i := strings.Index(s, ")"); i > 0 && i < len(s)-1 {
			s = strings.TrimSpace(s[i+1:])
		}
	}
	s = bracketNoteRe.ReplaceAllString(s, "")
	if sents := sentences(s); len(sents) > 0 {
		s = sents[0]
	}
	s = strings.TrimSpace(s)
	if n := len(s); n > 1 && (s[n-1] == '.' || s[n-1] == ';') && s[n-2] != ' ' && s[n-2] != '.' {
		s = s[:n-1]
	}
	if s == "" {
		name := o.Names[len(o.Names)-1]
		s = strings.ReplaceAll(strings.TrimLeft(name, "-"), "-", " ")
	}
	s = capitalize(s)
	if utf8.RuneCountInString(s) > 72 {
		s = string([]rune(s)[:71]) + "…"
	}
	return s
}
