package manpage

import (
	"regexp"
	"strings"
)

// Example is a ready-made command line from the manual, with what it does.
type Example struct {
	Line string `json:"line"`
	Desc string `json:"desc,omitempty"`
}

// maxExamples caps how many examples are kept from one manual.
const maxExamples = 40

var (
	exampleSects = map[string]bool{"EXAMPLES": true, "EXAMPLE": true, "USAGE": true}
	bulletRe     = regexp.MustCompile(`^(?:o|•|·|\*|-|\d+\.)\s+`)
)

// findExamples collects the example command lines of the manual's EXAMPLES
// section (or USAGE, or a --help "Examples:" heading). Only lines that run
// this command on their own, with no pipes or redirections, are kept, since
// only those can be loaded into the form. Each gets the explanation next to
// it: the indented text below it (git, BSD pages), "# comment" lines above
// it (kubectl), or the paragraph before or after it (GNU find, rsync).
func findExamples(lines []line, command, source string) []Example {
	var region []line
	for _, l := range lines {
		in := exampleSects[l.sect]
		if source == "help" {
			in = strings.HasPrefix(strings.ToLower(l.sub), "example")
		}
		switch {
		case !in || l.ind == -1:
		case l.ind == -2:
			region = append(region, line{}) // a subsection title separates paragraphs
		default:
			region = append(region, l)
		}
	}
	words := strings.Fields(command)
	var out []Example
	seen := map[string]bool{}
	style := "" // "precede" or "follow": where this manual puts explanations
	lastEnd, lastDesc := -2, ""
	for i := 0; i < len(region) && len(out) < maxExamples; i++ {
		cmd, ok := exampleAt(region, i, words)
		if !ok {
			continue
		}
		start := i
		// Join continuation lines ending in a backslash.
		for strings.HasSuffix(cmd, `\`) && i+1 < len(region) && !region[i+1].blank() {
			i++
			cmd = strings.TrimSpace(strings.TrimSuffix(cmd, `\`)) + " " + region[i].s
		}
		if style == "" {
			style = "follow"
			if p := paraBefore(region, start, words); p != "" && (!strings.HasSuffix(p, ":") || bulletRe.MatchString(p)) {
				style = "precede"
			}
		}
		var desc string
		if onlyBlanks(region, lastEnd+1, start) {
			desc = lastDesc // one explanation for a run of examples
		} else {
			desc = exampleDesc(region, start, i, words, style)
		}
		lastEnd, lastDesc = i, desc
		cmd = spacesRe.ReplaceAllString(cmd, " ")
		if seen[cmd] || hasShellOperator(cmd) {
			continue
		}
		seen[cmd] = true
		out = append(out, Example{Line: cmd, Desc: desc})
	}
	return out
}

// exampleAt returns the command on line i if it is an example of words.
func exampleAt(region []line, i int, words []string) (string, bool) {
	l := region[i]
	if l.blank() {
		return "", false
	}
	s := l.s
	for _, prompt := range []string{"$ ", "% "} {
		s = strings.TrimPrefix(s, prompt)
	}
	f := strings.Fields(s)
	if len(f) < len(words) {
		return "", false
	}
	for k, w := range words {
		if f[k] != w {
			return "", false
		}
	}
	// Prose that happens to start with the command's name ends like a
	// sentence and has no options.
	if strings.HasSuffix(s, ":") {
		return "", false
	}
	if strings.HasSuffix(s, ".") && len(f) > 4 && !strings.Contains(s, " -") {
		return "", false
	}
	// It must stand on its own: after a blank line, a comment, another
	// example, or indented deeper than the text it follows.
	if i > 0 {
		prev := region[i-1]
		_, prevEx := exampleAt(region, i-1, words)
		if !prev.blank() && !prevEx && !strings.HasPrefix(prev.s, "#") && prev.ind >= l.ind {
			return "", false
		}
	}
	return s, true
}

// exampleDesc finds the explanation for the example on lines start..end.
func exampleDesc(region []line, start, end int, words []string, style string) string {
	// "# List all pods" above it.
	var comments []string
	for j := start - 1; j >= 0 && strings.HasPrefix(region[j].s, "#"); j-- {
		comments = append([]string{strings.TrimSpace(strings.TrimLeft(region[j].s, "#"))}, comments...)
	}
	if len(comments) > 0 {
		return cleanDesc(strings.Join(comments, " "))
	}
	// Indented text right below it.
	ind := region[start].ind
	var below []string
	for j := end + 1; j < len(region) && !region[j].blank() && region[j].ind > ind; j++ {
		below = append(below, region[j].s)
	}
	if len(below) > 0 {
		return cleanDesc(strings.Join(below, " "))
	}
	before := paraBefore(region, start, words)
	if style == "precede" || bulletRe.MatchString(before) {
		return cleanDesc(before)
	}
	// Text after it, unless that ends in ":" and so introduces the next
	// example; then text before it that introduces this one.
	after := paraAfter(region, end, words)
	if after != "" && !strings.HasSuffix(after, ":") {
		return cleanDesc(after)
	}
	if strings.HasSuffix(before, ":") {
		return cleanDesc(before)
	}
	return cleanDesc(after) // explains this one and leads into the next
}

func onlyBlanks(region []line, from, to int) bool {
	if from < 0 || from > to {
		return false
	}
	for j := from; j < to; j++ {
		if !region[j].blank() {
			return false
		}
	}
	return true
}

// paraBefore returns the paragraph just before line i, unless it is
// another example.
func paraBefore(region []line, i int, words []string) string {
	j := i - 1
	for j >= 0 && region[j].blank() {
		j--
	}
	var parts []string
	for ; j >= 0 && !region[j].blank(); j-- {
		if _, ok := exampleAt(region, j, words); ok {
			return ""
		}
		if region[j].ind <= 4 && len(parts) > 0 && region[j].ind < region[j+1].ind {
			break // a heading above the paragraph
		}
		parts = append([]string{region[j].s}, parts...)
	}
	return strings.Join(parts, " ")
}

// paraAfter returns the paragraph just after line i, unless it is another
// example.
func paraAfter(region []line, i int, words []string) string {
	j := i + 1
	for j < len(region) && region[j].blank() {
		j++
	}
	var parts []string
	for ; j < len(region) && !region[j].blank(); j++ {
		if _, ok := exampleAt(region, j, words); ok {
			return strings.Join(parts, " ")
		}
		parts = append(parts, region[j].s)
	}
	return strings.Join(parts, " ")
}

func cleanDesc(s string) string {
	s = strings.TrimSpace(bulletRe.ReplaceAllString(strings.TrimSpace(s), ""))
	s = strings.TrimRight(s, " :") // "…in a file:" introduced the example
	return spacesRe.ReplaceAllString(s, " ")
}

// hasShellOperator reports whether s pipes, chains or redirects, outside
// quotes: such lines aren't a single run of the command.
func hasShellOperator(s string) bool {
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else if c == '\\' && quote == '"' {
				i++
			}
		case c == '\\':
			i++
		case c == '\'' || c == '"':
			quote = c
		case strings.IndexByte("|;&<>`", c) >= 0:
			return true
		case c == '$' && i+1 < len(s) && s[i+1] == '(':
			return true
		}
	}
	return false
}

// optionIntroRe matches the line that introduces an option's own examples:
// "Example:", "Examples:", "For example:".
var optionIntroRe = regexp.MustCompile(`(?i)(?:^|\s)(?:for )?examples?:$`)

// optionExamples reads the command lines given under an option, after a
// line ending in "Example:" (curl puts one under nearly every option).
func optionExamples(block []line, command string) []string {
	words := strings.Fields(command)
	var out []string
	seen := map[string]bool{}
	for i := 0; i < len(block); i++ {
		if !optionIntroRe.MatchString(block[i].s) {
			continue
		}
		ind := block[i].ind
		for j := i + 1; j < len(block); j++ {
			l := block[j]
			if l.blank() {
				if len(out) > 0 && j+1 < len(block) && block[j+1].ind <= ind {
					break
				}
				continue
			}
			if l.ind <= ind {
				break
			}
			cmd, ok := exampleAt(block, j, words)
			if !ok {
				continue
			}
			for strings.HasSuffix(cmd, `\`) && j+1 < len(block) && !block[j+1].blank() {
				j++
				cmd = strings.TrimSpace(strings.TrimSuffix(cmd, `\`)) + " " + block[j].s
			}
			cmd = spacesRe.ReplaceAllString(cmd, " ")
			if !seen[cmd] {
				seen[cmd] = true
				out = append(out, cmd)
			}
			i = j
		}
	}
	return out
}
