package manpage

import (
	"os"
	"strings"

	"github.com/lonedevel/commando/internal/complete"
)

// ApplyCompletions turns text options into dropdowns using the value lists
// from the command's shell completion definitions. Lists already inferred
// from the manual are kept, after the completion's values. When the manual
// has its own list, a completion value it never mentions is left out:
// completion files are sometimes wrong (fish offers ls --sort=atime, which
// GNU ls rejects).
func ApplyCompletions(s *Spec, res complete.Result) {
	if len(res.Values) == 0 {
		return
	}
	for i := range s.Options {
		o := &s.Options[i]
		if !o.TakesArg() || (o.Kind != KindString && o.Kind != KindChoice) {
			continue
		}
		var vals []string
		var src string
		for _, n := range o.Names {
			if v := res.Values[n]; len(v) >= 2 {
				vals, src = v, res.Sources[n]
				break
			}
		}
		if vals == nil {
			continue
		}
		if len(o.Choices) >= 2 {
			var kept []string
			for _, v := range vals {
				if indexOfString(o.Choices, v) >= 0 || mentions(o.Desc, v) {
					kept = append(kept, v)
				}
			}
			vals = kept
			if len(vals) == 0 {
				continue
			}
		}
		merged := append([]string(nil), vals...)
		for _, c := range o.Choices {
			if indexOfString(merged, c) < 0 {
				merged = append(merged, c)
			}
		}
		o.Kind = KindChoice
		o.Choices = merged
		o.ChoiceSource = src
	}
}

// mentions reports whether text has word w, not as part of a longer word.
func mentions(text, w string) bool {
	for i := 0; ; {
		j := strings.Index(text[i:], w)
		if j < 0 {
			return false
		}
		j += i
		before := j == 0 || !isWordByte(text[j-1])
		after := j+len(w) == len(text) || !isWordByte(text[j+len(w)])
		if before && after {
			return true
		}
		i = j + 1
	}
}

func isWordByte(c byte) bool {
	return c == '-' || c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

func indexOfString(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return -1
}

// completionsFor looks up shell completion values for a single command,
// unless COMMANDO_NO_COMPLETIONS is set. Subcommands ("git commit") are
// skipped: their definitions are conditional and would be misattributed.
func completionsFor(words []string) complete.Result {
	if len(words) != 1 || os.Getenv("COMMANDO_NO_COMPLETIONS") != "" {
		return complete.Result{}
	}
	return complete.Lookup(words[0])
}
