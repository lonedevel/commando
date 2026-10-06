// Package cmdline turns form state into a shell command line and back.
package cmdline

import (
	"regexp"
	"sort"
	"strings"

	"github.com/lonedevel/commando/internal/manpage"
)

// Value is the state of one option in the form.
type Value struct {
	On    bool   // flag set / option present
	Count int    // repeat count for repeatable flags (>=1 when On)
	Text  string // argument value
	Seq   int    // when it was turned on, to keep an expression's order (find)
}

// TokenKind classifies tokens for syntax highlighting.
type TokenKind int

const (
	TokCommand TokenKind = iota
	TokFlag
	TokValue
	TokArg
)

// Token is one piece of the rendered command line.
type Token struct {
	Text   string
	Kind   TokenKind
	Attach bool // no space before this token ("--width=" "80")
}

// Render returns the plain command string.
func Render(toks []Token) string {
	var b strings.Builder
	for i, t := range toks {
		if i > 0 && !t.Attach {
			b.WriteByte(' ')
		}
		b.WriteString(t.Text)
	}
	return b.String()
}

var safeRe = regexp.MustCompile(`^[A-Za-z0-9_@%+=:,./~^-]+$`)

// Quote quotes s for a POSIX shell when needed.
func Quote(s string) string {
	if s == "" {
		return "''"
	}
	if safeRe.MatchString(s) {
		return s
	}
	// Keep "~/" expandable.
	if strings.HasPrefix(s, "~/") && safeRe.MatchString("~/") {
		return "~/" + Quote(s[2:])
	}
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// Build renders the command line. values is indexed like spec.Options.
// args is appended verbatim (it is shell text typed by the user).
// preferLong chooses --long names over -s short names where both exist.
func Build(spec *manpage.Spec, values []Value, args string, preferLong bool) []Token {
	toks := []Token{{Text: spec.Command, Kind: TokCommand}}
	// In an expression (find), tests and actions follow the paths, in the
	// order they were turned on.
	expr := spec.Expression()
	primary := func(i int) bool { return expr && spec.Options[i].Primary() }

	// Cluster short boolean flags: -la
	cluster := ""
	clustered := make([]bool, len(values))
	if !preferLong {
		for i, v := range values {
			o := &spec.Options[i]
			if !v.On || o.TakesArg() || primary(i) {
				continue
			}
			if n := shortName(o); len(n) == 2 && isClusterable(n[1]) {
				cluster += strings.Repeat(n[1:], max(1, v.Count))
				clustered[i] = true
			}
		}
		if cluster != "" {
			toks = append(toks, Token{Text: "-" + cluster, Kind: TokFlag})
		}
	}

	var primaries []int
	for i, v := range values {
		if !v.On || clustered[i] {
			continue
		}
		if primary(i) {
			primaries = append(primaries, i)
			continue
		}
		toks = appendOption(toks, &spec.Options[i], v, preferLong)
	}
	if a := strings.TrimSpace(args); a != "" {
		toks = append(toks, Token{Text: a, Kind: TokArg})
	}
	sortExpression(spec, values, primaries)
	for _, i := range primaries {
		toks = appendOption(toks, &spec.Options[i], values[i], preferLong)
	}
	return toks
}

// sortExpression orders an expression's tests and actions: in the order
// they were turned on, but actions after the tests that pick the files, as
// find . -delete -name x would delete everything.
func sortExpression(spec *manpage.Spec, values []Value, primaries []int) {
	sort.SliceStable(primaries, func(a, b int) bool {
		if aa, ab := isAction(&spec.Options[primaries[a]]), isAction(&spec.Options[primaries[b]]); aa != ab {
			return ab
		}
		sa, sb := values[primaries[a]].Seq, values[primaries[b]].Seq
		if sa == 0 || sb == 0 {
			return sa != 0 && sb == 0 // ordered ones first
		}
		return sa < sb
	})
}

// findActions are find's actions, which act on the files its tests pick.
var findActions = map[string]bool{
	"-delete": true, "-exec": true, "-execdir": true, "-ok": true, "-okdir": true,
	"-print": true, "-print0": true, "-printf": true, "-fprint": true, "-fprint0": true,
	"-fprintf": true, "-ls": true, "-fls": true, "-quit": true,
}

func isAction(o *manpage.Option) bool {
	for _, n := range o.Names {
		if findActions[n] {
			return true
		}
	}
	return false
}

// appendOption renders one set option and its value.
func appendOption(toks []Token, o *manpage.Option, v Value, preferLong bool) []Token {
	name := pickName(o, preferLong)
	if !o.TakesArg() {
		for c := 0; c < max(1, v.Count); c++ {
			toks = append(toks, Token{Text: name, Kind: TokFlag})
		}
		return toks
	}
	if v.Text == "" {
		if o.ArgOptional {
			toks = append(toks, Token{Text: name, Kind: TokFlag})
		}
		return toks
	}
	val := Quote(v.Text)
	switch {
	case strings.HasPrefix(name, "--") && (o.LongEquals || o.ArgOptional):
		return append(toks, Token{Text: name + "=", Kind: TokFlag}, Token{Text: val, Kind: TokValue, Attach: true})
	case !strings.HasPrefix(name, "--") && o.ArgOptional:
		return append(toks, Token{Text: name, Kind: TokFlag}, Token{Text: val, Kind: TokValue, Attach: true})
	default:
		return append(toks, Token{Text: name, Kind: TokFlag}, Token{Text: val, Kind: TokValue})
	}
}

func isClusterable(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '@' || c == '%' || c == ','
}

func shortName(o *manpage.Option) string {
	for _, n := range o.Names {
		if !strings.HasPrefix(n, "--") {
			return n
		}
	}
	return ""
}

func pickName(o *manpage.Option, preferLong bool) string {
	s, l := o.Short(), o.Long()
	if preferLong && len(l) > 0 {
		return l[0]
	}
	if len(s) > 0 {
		return s[0]
	}
	return l[0]
}
