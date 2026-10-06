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
	values, rest, _ := scan(spec, words)
	return values, strings.Join(rest, " ")
}

// item is one option (opt >= 0) or positional word (opt < 0), in the order
// it appeared on the command line.
type item struct {
	opt        int
	raw        string // as written: "-f backup.tgz", "--exclude=.git", "src"
	val        string // the option's value, or the word's, unquoted
	positional bool   // after "--": an argument whatever it looks like
}

// Faithful reports whether the form can hold words exactly: rebuilding
// them gives the same command. That isn't so when an option follows the
// positional arguments (git log master --not --remotes), when an option
// with a value or two choices of one group appear twice, or, for an
// expression (find), when anything but its tests and actions follows the
// paths (!, parentheses, -exec … \;).
func Faithful(spec *manpage.Spec, words []Word) bool {
	values, _, items := scan(spec, words)
	expr := spec.Expression()
	seen := map[int]bool{}
	groups := map[int]int{} // group → the option chosen
	stage := 0              // 0: options, 1: positional words, 2: an expression's tests and actions
	for _, it := range items {
		if it.opt < 0 {
			if stage == 2 {
				return false
			}
			stage = max(stage, 1)
			continue
		}
		o := &spec.Options[it.opt]
		if seen[it.opt] && (o.TakesArg() || !o.Repeatable) {
			return false
		}
		seen[it.opt] = true
		if g := spec.GroupOf(it.opt); g >= 0 {
			if prev, ok := groups[g]; ok && prev != it.opt {
				return false // the form keeps one choice of a group
			}
			groups[g] = it.opt
		}
		switch {
		case expr && o.Primary():
			stage = 2
		case stage > 0:
			return false
		}
	}
	if expr {
		// The form moves actions after tests; the line must already be so.
		var typed []int
		for _, it := range items {
			if it.opt >= 0 && spec.Options[it.opt].Primary() {
				typed = append(typed, it.opt)
			}
		}
		built := append([]int(nil), typed...)
		sortExpression(spec, values, built)
		for k := range typed {
			if typed[k] != built[k] {
				return false
			}
		}
	}
	return true
}

func scan(spec *manpage.Spec, words []Word) (values []Value, rest []string, items []item) {
	values = make([]Value, len(spec.Options))
	byName := map[string]int{}
	for i, o := range spec.Options {
		for _, n := range o.Names {
			byName[n] = i
		}
	}
	set := func(i int, text, raw string) {
		v := &values[i]
		items = append(items, item{opt: i, raw: raw, val: text})
		if v.On && !spec.Options[i].TakesArg() && spec.Options[i].Repeatable {
			v.Count++
		} else {
			if !v.On {
				v.Seq = len(items)
			}
			v.On, v.Count = true, 1
		}
		if text != "" {
			v.Text = text
		}
	}
	addRest := func(w Word, positional bool) {
		rest = append(rest, w.Raw)
		items = append(items, item{opt: -1, raw: w.Raw, val: w.Value, positional: positional})
	}
	for k := 0; k < len(words); k++ {
		w := words[k]
		s := w.Value
		next := func() (string, string, bool) {
			if k+1 < len(words) {
				k++
				return words[k].Value, words[k].Raw, true
			}
			return "", "", false
		}
		switch {
		case s == "--":
			addRest(w, false)
			for _, r := range words[k+1:] {
				addRest(r, true)
			}
			k = len(words)
		case strings.HasPrefix(s, "--") || (strings.HasPrefix(s, "-") && len(s) > 2 && hasExact(byName, s)):
			name, val, hasEq := strings.Cut(s, "=")
			i, ok := byName[name]
			if !ok {
				addRest(w, false)
				continue
			}
			o := &spec.Options[i]
			raw := w.Raw
			if o.TakesArg() && !hasEq && !o.ArgOptional {
				var vraw string
				if val, vraw, ok = next(); ok {
					raw += " " + vraw
				}
			}
			set(i, val, raw)
		case strings.HasPrefix(s, "-") && len(s) > 1:
			if !applyCluster(spec, byName, w.Raw, s, set, next) {
				addRest(w, false)
			}
		default:
			addRest(w, false)
		}
	}
	return values, rest, items
}

func hasExact(m map[string]int, s string) bool {
	name, _, _ := strings.Cut(s, "=")
	_, ok := m[name]
	return ok
}

// applyCluster handles "-la", "-n5", "-C 3". It only applies anything when
// every letter is known, so unknown words pass through untouched.
func applyCluster(spec *manpage.Spec, byName map[string]int, raw, s string, set func(int, string, string), next func() (string, string, bool)) bool {
	type hit struct {
		i        int
		val, raw string
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
				hits = append(hits, hit{i, s[j+1:], s[j:]})
			} else if !o.ArgOptional {
				needNext = len(hits)
				hits = append(hits, hit{i, "", s[j:]})
			} else {
				hits = append(hits, hit{i, "", s[j:]})
			}
			break
		}
		hits = append(hits, hit{i, "", s[j : j+1]})
	}
	if needNext >= 0 {
		if v, vraw, ok := next(); ok {
			hits[needNext].val = v
			hits[needNext].raw += " " + vraw
		}
	}
	for _, h := range hits {
		r := "-" + h.raw
		if len(hits) == 1 && needNext < 0 {
			r = raw // as typed, quotes and all
		}
		set(h.i, h.val, r)
	}
	return true
}
