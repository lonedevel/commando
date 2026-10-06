package cmdline

import (
	"strings"

	"github.com/lonedevel/commando/internal/manpage"
)

// PartKind classifies one piece of an explained command line.
type PartKind int

const (
	PartOption   PartKind = iota // an option the manual documents
	PartArg                      // a positional argument
	PartUnknown                  // looks like an option, but the manual doesn't list it
	PartOperator                 // an expression operator, as in find: ! ( ) -o
	PartEnd                      // "--": the end of the options
)

// Part is one piece of a command line, in the order it was written.
type Part struct {
	Text  string // as written: "-f backup.tgz", "--exclude=.git", "src"
	Kind  PartKind
	Opt   int    // index into spec.Options, for PartOption
	Value string // the option's value, unquoted
	Arg   string // the argument's name from the usage line ("SOURCE"), for PartArg
}

// Operators of find-style expressions.
var exprOperators = map[string]bool{
	"!": true, "(": true, ")": true, "-o": true, "-or": true, "-a": true,
	"-and": true, "-not": true, ",": true,
}

// Explain splits words (after the command itself) into the options and
// arguments the manual knows them as, keeping the order they were written.
func Explain(spec *manpage.Spec, words []Word) []Part {
	_, _, items := scan(spec, words)
	expr := spec.Expression()
	parts := make([]Part, 0, len(items))
	npos := 0
	for _, it := range items {
		switch {
		case it.opt >= 0:
			parts = append(parts, Part{Text: it.raw, Kind: PartOption, Opt: it.opt, Value: it.val})
		case it.positional:
			parts = append(parts, Part{Text: it.raw, Kind: PartArg})
			npos++
		case it.raw == "--":
			parts = append(parts, Part{Text: it.raw, Kind: PartEnd})
		default:
			v := it.val
			switch {
			case expr && exprOperators[v]:
				parts = append(parts, Part{Text: it.raw, Kind: PartOperator, Value: v})
			case strings.HasPrefix(v, "-") && len(v) > 1:
				parts = append(parts, Part{Text: it.raw, Kind: PartUnknown})
			default:
				parts = append(parts, Part{Text: it.raw, Kind: PartArg})
				npos++
			}
		}
	}
	// Name the arguments after the usage line's. An expression (find's)
	// is made of options, so it doesn't name a word.
	args := spec.Args
	if expr {
		args = nil
		for _, a := range spec.Args {
			if !strings.EqualFold(a.Name, "expression") {
				args = append(args, a)
			}
		}
	}
	names := argNames(args, npos)
	k := 0
	for i := range parts {
		if parts[i].Kind == PartArg {
			parts[i].Arg = names[k]
			k++
		}
	}
	return parts
}

// argNames gives n positional words the names of the usage line's
// arguments, in order: single ones take a word each, a repeatable one
// takes what the ones after it don't need. Words beyond the usage line
// get no name.
func argNames(args []manpage.Arg, n int) []string {
	out := make([]string, n)
	k := 0
	for i, a := range args {
		take := 1
		if a.Repeat {
			after := 0
			for _, b := range args[i+1:] {
				if !b.Repeat {
					after++
				}
			}
			take = max(0, n-k-after)
		}
		for ; take > 0 && k < n; take-- {
			out[k] = a.Name
			k++
		}
	}
	if k < n && len(args) > 0 && args[len(args)-1].Repeat {
		for ; k < n; k++ {
			out[k] = args[len(args)-1].Name
		}
	}
	return out
}
