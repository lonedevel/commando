package manpage

import (
	"os"

	"github.com/lonedevel/commando/internal/complete"
)

// ApplyCompletions turns text options into dropdowns using the value lists
// from the command's shell completion definitions. Lists already inferred
// from the manual are kept, after the completion's values.
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
