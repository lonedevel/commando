// Package manpage locates, renders and parses Unix manual pages (or --help
// output) into a structured description of a command's options.
package manpage

import (
	"regexp"
	"strings"
)

// Kind describes what sort of value an option takes, which decides the
// control used to edit it.
type Kind int

const (
	KindFlag   Kind = iota // boolean switch: checkbox
	KindString             // free text: text input
	KindNumber             // numeric: text input restricted to digits
	KindPath               // file or directory: text input with completion
	KindChoice             // enumerated values: dropdown
)

func (k Kind) String() string {
	switch k {
	case KindFlag:
		return "flag"
	case KindString:
		return "text"
	case KindNumber:
		return "number"
	case KindPath:
		return "path"
	case KindChoice:
		return "choice"
	}
	return "unknown"
}

// Option is a single command-line option.
type Option struct {
	Names       []string `json:"names"`                  // e.g. ["-a", "--all"]
	Arg         string   `json:"arg,omitempty"`          // placeholder, e.g. "WHEN"
	ArgOptional bool     `json:"arg_optional,omitempty"` // --color[=WHEN]
	LongEquals  bool     `json:"long_equals,omitempty"`  // long form written as --name=ARG
	Kind        Kind     `json:"kind"`
	Choices     []string `json:"choices,omitempty"`
	// Examples are command lines the manual gives under this option
	// ("Example:\n  curl --retry 7 https://example.com").
	Examples []string `json:"examples,omitempty"`
	// ChoiceDesc says what each value means, where the manual does.
	ChoiceDesc map[string]string `json:"choice_desc,omitempty"`
	// ChoiceSource is "zsh" or "fish" when Choices came from that shell's
	// completion definitions, empty when inferred from the manual.
	ChoiceSource string   `json:"choice_source,omitempty"`
	Repeatable   bool     `json:"repeatable,omitempty"` // e.g. -v -v -v
	Label        string   `json:"label"`                // short human label
	Desc         string   `json:"desc"`                 // full description, paragraphs split by "\n\n"
	Section      string   `json:"section,omitempty"`    // man (sub)section it was found in
	Conflicts    []string `json:"conflicts,omitempty"`  // options turned off when this one is set
	Notes        []string `json:"notes,omitempty"`      // related general paragraphs ("The WHEN argument ...")
	Danger       string   `json:"danger,omitempty"`     // why the option is risky, e.g. "can delete or overwrite data"
}

// Group is a set of mutually exclusive boolean options, shown as radio buttons.
type Group struct {
	Label   string `json:"label"`
	Members []int  `json:"members"` // indexes into Spec.Options
}

// Spec is everything commando knows about a command.
type Spec struct {
	Command     string    `json:"command"`
	Summary     string    `json:"summary"`
	Synopsis    string    `json:"synopsis"`
	Description string    `json:"description"`
	Options     []Option  `json:"options"`
	Args        []Arg     `json:"args,omitempty"` // positional arguments; nil means use one free-text field
	Groups      []Group   `json:"groups,omitempty"`
	Examples    []Example `json:"examples,omitempty"` // ready-made command lines from the manual
	Manual      string    `json:"manual"`             // cleaned full text, for the manual viewer
	Source      string    `json:"source"`             // "man" or "help"
	Page        string    `json:"page,omitempty"`     // the manual page file, when Source is "man"
}

// Short returns the option's single-dash names.
func (o *Option) Short() []string {
	var out []string
	for _, n := range o.Names {
		if !strings.HasPrefix(n, "--") {
			out = append(out, n)
		}
	}
	return out
}

// Long returns the option's double-dash names.
func (o *Option) Long() []string {
	var out []string
	for _, n := range o.Names {
		if strings.HasPrefix(n, "--") {
			out = append(out, n)
		}
	}
	return out
}

// TakesArg reports whether the option accepts a value.
func (o *Option) TakesArg() bool { return o.Arg != "" }

// Primary reports whether o is a test or action of an expression, as in
// find (-name, -type, -delete): every name is a single-dash word.
func (o *Option) Primary() bool {
	for _, n := range o.Names {
		if strings.HasPrefix(n, "--") || len(n) <= 2 {
			return false
		}
	}
	return len(o.Names) > 0
}

var expressionRe = regexp.MustCompile(`(?i)\bexpression\b`)

// Expression reports whether the command takes an expression after its
// operands, as find does ("find [-H] path ... [expression]"). Its tests
// and actions then go after the paths, and their order matters.
func (s *Spec) Expression() bool { return expressionRe.MatchString(s.Synopsis) }

// HasName reports whether name is one of the option's spellings.
func (o *Option) HasName(name string) bool {
	for _, n := range o.Names {
		if n == name {
			return true
		}
	}
	return false
}

// GroupOf returns the index of the group containing option i, or -1.
func (s *Spec) GroupOf(i int) int {
	for g, grp := range s.Groups {
		for _, m := range grp.Members {
			if m == i {
				return g
			}
		}
	}
	return -1
}
