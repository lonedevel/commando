package cmdline

import (
	"strings"

	"github.com/lonedevel/commando/internal/manpage"
	"github.com/lonedevel/commando/internal/shellword"
)

// Word is a shell word: Raw is the text as typed, Value has quotes removed.
type Word = shellword.Word

// Split splits a command line into shell words; see shellword.Split.
func Split(line string) []Word { return shellword.Split(line) }

// Prefill maps existing command-line words (after the command itself) onto
// form values. Words it cannot attribute to a known option are returned as
// the positional argument text, preserving how they were typed.
func Prefill(spec *manpage.Spec, words []Word) (values []Value, args string) {
	values = make([]Value, len(spec.Options))
	byName := map[string]int{}
	for i, o := range spec.Options {
		for _, n := range o.Names {
			byName[n] = i
		}
	}
	set := func(i int, text string) {
		v := &values[i]
		if v.On && !spec.Options[i].TakesArg() && spec.Options[i].Repeatable {
			v.Count++
		} else {
			v.On, v.Count = true, 1
		}
		if text != "" {
			v.Text = text
		}
	}
	var rest []string
	for k := 0; k < len(words); k++ {
		w := words[k]
		s := w.Value
		next := func() (string, bool) {
			if k+1 < len(words) {
				k++
				return words[k].Value, true
			}
			return "", false
		}
		switch {
		case s == "--":
			for _, r := range words[k:] {
				rest = append(rest, r.Raw)
			}
			k = len(words)
		case strings.HasPrefix(s, "--") || (strings.HasPrefix(s, "-") && len(s) > 2 && hasExact(byName, s)):
			name, val, hasEq := strings.Cut(s, "=")
			i, ok := byName[name]
			if !ok {
				rest = append(rest, w.Raw)
				continue
			}
			o := &spec.Options[i]
			if o.TakesArg() && !hasEq && !o.ArgOptional {
				val, _ = next()
			}
			set(i, val)
		case strings.HasPrefix(s, "-") && len(s) > 1:
			if !applyCluster(spec, byName, s, set, next) {
				rest = append(rest, w.Raw)
			}
		default:
			rest = append(rest, w.Raw)
		}
	}
	return values, strings.Join(rest, " ")
}

func hasExact(m map[string]int, s string) bool {
	name, _, _ := strings.Cut(s, "=")
	_, ok := m[name]
	return ok
}

// applyCluster handles "-la", "-n5", "-C 3". It only applies anything when
// every letter is known, so unknown words pass through untouched.
func applyCluster(spec *manpage.Spec, byName map[string]int, s string, set func(int, string), next func() (string, bool)) bool {
	type hit struct {
		i   int
		val string
	}
	var hits []hit
	needNext := -1
	for j := 1; j < len(s); j++ {
		i, ok := byName["-"+s[j:j+1]]
		if !ok {
			return false
		}
		o := &spec.Options[i]
		if o.TakesArg() {
			if j+1 < len(s) {
				hits = append(hits, hit{i, s[j+1:]})
			} else if !o.ArgOptional {
				needNext = len(hits)
				hits = append(hits, hit{i, ""})
			} else {
				hits = append(hits, hit{i, ""})
			}
			break
		}
		hits = append(hits, hit{i, ""})
	}
	if needNext >= 0 {
		v, _ := next()
		hits[needNext].val = v
	}
	for _, h := range hits {
		set(h.i, h.val)
	}
	return true
}
