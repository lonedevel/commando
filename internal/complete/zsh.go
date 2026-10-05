package complete

import (
	"regexp"
	"strings"
)

var (
	// A value list at the end of a spec: ":message:(a b c)".
	zshListRe = regexp.MustCompile(`:\(([^()$]*)\)\s*$`)
	// A described value list: ":message:((a\:"desc" b\:"desc"))".
	zshDescListRe = regexp.MustCompile(`:\(\((.*)\)\)\s*$`)
	zshDescItemRe = regexp.MustCompile(`(?:^|\s)([A-Za-z0-9][A-Za-z0-9_.+@%/-]*)\\:`)
	zshBraceRe    = regexp.MustCompile(`\{([^{}]*)\}`)
)

// ParseZsh extracts option value lists from a zsh completion function. It
// reads _arguments specs line by line and only keeps fixed lists, ignoring
// specs whose values come from functions (":file:_files").
func ParseZsh(src string) Values {
	out := Values{}
	for _, line := range strings.Split(src, "\n") {
		spec := unquoteZsh(strings.TrimSpace(strings.TrimSuffix(strings.TrimSpace(line), "\\")))
		if spec == "" || strings.HasPrefix(spec, "#") || !strings.Contains(spec, ":(") {
			continue
		}
		var values []string
		if m := zshDescListRe.FindStringSubmatch(spec); m != nil {
			for _, it := range zshDescItemRe.FindAllStringSubmatch(m[1], -1) {
				values = append(values, it[1])
			}
		} else if m := zshListRe.FindStringSubmatch(spec); m != nil {
			for _, it := range strings.Fields(m[1]) {
				it = strings.ReplaceAll(it, `\`, "")
				if !validValue(it) {
					values = nil
					break
				}
				values = append(values, it)
			}
		}
		if len(values) < 2 {
			continue
		}
		for _, name := range zshNames(spec) {
			if _, seen := out[name]; !seen {
				out[name] = values
			}
		}
	}
	return out
}

// unquoteZsh removes shell quoting, keeping the text: '--a'{x,y}'[d]' →
// --a{x,y}[d]. Backslash escapes inside double quotes are left alone; they
// only matter in descriptions.
func unquoteZsh(s string) string {
	var b strings.Builder
	quote := byte(0)
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0 && c == quote:
			quote = 0
		case quote == 0 && (c == '\'' || c == '"'):
			quote = c
		default:
			b.WriteByte(c)
		}
	}
	return b.String()
}

// zshNames returns the option names a spec applies to, expanding braces:
// "(-a --all)"{-a,--all}"[...]" → -a, --all.
func zshNames(spec string) []string {
	s := strings.TrimLeft(spec, "*")
	// Drop a leading exclusion list "(-a -b)".
	if strings.HasPrefix(s, "(") {
		if i := strings.Index(s, ")"); i > 0 {
			s = s[i+1:]
		}
	}
	s = strings.TrimLeft(s, "*")
	// The names end where the description or the first colon begins.
	if i := strings.IndexAny(s, "[:"); i >= 0 {
		s = s[:i]
	}
	var names []string
	for _, n := range expandBraces(s) {
		// Strip argument markers: --opt=- --opt= -o+ -o- --opt=*
		n = strings.TrimRight(n, "=+-*")
		if i := strings.Index(n, "="); i > 0 {
			n = n[:i]
		}
		if n = strings.TrimRight(n, "*"); strings.HasPrefix(n, "-") && len(n) > 1 && !strings.ContainsAny(n, " ()$") {
			names = append(names, n)
		}
	}
	return names
}

func expandBraces(s string) []string {
	m := zshBraceRe.FindStringSubmatchIndex(s)
	if m == nil {
		return []string{s}
	}
	pre, body, post := s[:m[0]], s[m[2]:m[3]], s[m[1]:]
	var out []string
	for _, alt := range strings.Split(body, ",") {
		out = append(out, expandBraces(pre+alt+post)...)
	}
	return out
}
