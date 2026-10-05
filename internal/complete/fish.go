package complete

import (
	"strings"

	"github.com/lonedevel/commando/internal/shellword"
)

// ParseFish extracts option value lists from a fish completion file:
// `complete -c ls -l color -a "never always auto"`. Entries with a -n
// condition are skipped (they usually belong to a subcommand), as are
// value lists computed by a command substitution.
func ParseFish(src, command string) Values {
	out := Values{}
	for _, stmt := range fishStatements(src) {
		words := shellword.Split(stmt)
		for len(words) > 0 && (words[0].Value == "and" || words[0].Value == "or" || words[0].Value == "not") {
			words = words[1:]
		}
		if len(words) == 0 || words[0].Value != "complete" {
			continue
		}
		e := parseComplete(words[1:])
		if e.cond || e.args == "" || !contains(e.commands, command) {
			continue
		}
		values := fishValues(e.args)
		if len(values) < 2 {
			continue
		}
		for _, n := range e.names {
			if _, seen := out[n]; !seen {
				out[n] = values
			}
		}
	}
	return out
}

type fishEntry struct {
	commands []string
	names    []string
	args     string
	cond     bool
}

func parseComplete(words []shellword.Word) fishEntry {
	var e fishEntry
	takes := map[byte]bool{'c': true, 's': true, 'l': true, 'o': true, 'a': true, 'd': true, 'n': true, 'w': true, 'p': true, 'C': true}
	set := func(flag byte, val string) {
		switch flag {
		case 'c':
			e.commands = append(e.commands, val)
		case 's':
			e.names = append(e.names, "-"+val)
		case 'l':
			e.names = append(e.names, "--"+val)
		case 'o':
			e.names = append(e.names, "-"+val)
		case 'a':
			e.args = val
		case 'n':
			e.cond = true
		}
	}
	long := map[string]byte{
		"--command": 'c', "--short-option": 's', "--long-option": 'l', "--old-option": 'o',
		"--arguments": 'a', "--description": 'd', "--condition": 'n', "--wraps": 'w', "--path": 'p',
	}
	for i := 0; i < len(words); i++ {
		w := words[i].Value
		if strings.HasPrefix(w, "--") {
			name, val, hasVal := strings.Cut(w, "=")
			f, ok := long[name]
			if !ok {
				continue
			}
			if !hasVal && i+1 < len(words) {
				i++
				val = words[i].Value
			}
			set(f, val)
			continue
		}
		if !strings.HasPrefix(w, "-") || len(w) < 2 {
			continue
		}
		// Clustered short flags: -xa "values", -fa, -sX.
		for j := 1; j < len(w); j++ {
			f := w[j]
			if !takes[f] {
				continue
			}
			val := w[j+1:]
			if val == "" && i+1 < len(words) {
				i++
				val = words[i].Value
			}
			set(f, val)
			break
		}
	}
	return e
}

// fishValues turns an -a argument into values. Items may carry a
// description after a tab ("size\tSort by size").
func fishValues(args string) []string {
	if strings.ContainsAny(args, "($") {
		return nil
	}
	// Mark the value/description separator before tokenizing, which would
	// otherwise read the backslash in "size\tSort by size" as an escape.
	args = strings.NewReplacer(`\t`, "\x00", "\t", " ", "\n", " ").Replace(args)
	var out []string
	for _, w := range shellword.Split(args) {
		v := w.Value
		if i := strings.IndexByte(v, 0); i >= 0 {
			v = v[:i]
		}
		v = strings.TrimRight(v, ",")
		if v == "" {
			continue
		}
		if !validValue(v) {
			return nil
		}
		out = append(out, v)
	}
	return out
}

// fishStatements splits fish source into statements, joining lines inside
// quotes and dropping comments.
func fishStatements(src string) []string {
	var out []string
	var b strings.Builder
	quote := byte(0)
	atWordStart := true
	flush := func() {
		if s := strings.TrimSpace(b.String()); s != "" {
			out = append(out, s)
		}
		b.Reset()
	}
	for i := 0; i < len(src); i++ {
		c := src[i]
		switch {
		case quote != 0:
			if c == '\\' && i+1 < len(src) {
				b.WriteByte(c)
				i++
				b.WriteByte(src[i])
				continue
			}
			if c == quote {
				quote = 0
			}
			b.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			b.WriteByte(c)
		case c == '\\' && i+1 < len(src) && src[i+1] == '\n':
			i++ // line continuation
			b.WriteByte(' ')
		case c == '\\' && i+1 < len(src):
			b.WriteByte(c)
			i++
			b.WriteByte(src[i])
		case c == '#' && atWordStart:
			for i < len(src) && src[i] != '\n' {
				i++
			}
			flush()
		case c == '\n' || c == ';':
			flush()
		default:
			b.WriteByte(c)
		}
		atWordStart = c == ' ' || c == '\t' || c == '\n' || c == ';'
	}
	flush()
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
