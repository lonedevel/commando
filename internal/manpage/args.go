package manpage

import (
	"regexp"
	"strings"
	"unicode"
)

// Arg is a positional argument from the command's usage line, e.g. SOURCE
// and DEST in "cp [OPTION]... SOURCE DEST".
type Arg struct {
	Name     string `json:"name"`               // as written: "SOURCE", "file", "LINK_NAME"
	Required bool   `json:"required,omitempty"` // not in [brackets]
	Repeat   bool   `json:"repeat,omitempty"`   // "FILE..." or "[file ...]"
	Path     bool   `json:"path,omitempty"`     // names a file or directory
}

// Label is a readable name: "LINK_NAME" → "Link name".
func (a Arg) Label() string {
	s := strings.ToLower(strings.NewReplacer("_", " ", "-", " ").Replace(a.Name))
	return capitalize(s)
}

const maxArgs = 5

var (
	argNameRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.:-]*$`)
	argPathRe = regexp.MustCompile(`(?i)(file|dir|path|source|src|dest|target|link|archive|starting-point)`)
)

// parseArgs reads the positional arguments from the first usage line. It
// returns nil when the line is too irregular to trust ("{A|c|x}...",
// "[options / URLs]", nested argument groups), so the form falls back to a
// single free-text Arguments field.
func parseArgs(synopsis, command string) []Arg {
	line := strings.TrimSpace(strings.SplitN(synopsis, "\n", 2)[0])
	if l := strings.ToLower(line); strings.HasPrefix(l, "usage:") {
		line = strings.TrimSpace(line[len("usage:"):])
	}
	toks := synopsisTokens(line)
	// The usage line must start with the command itself ("git commit" or
	// "git-commit"); "Local: rsync ..." and the like are not trusted.
	cmdWords := strings.Fields(command)
	switch {
	case len(toks) >= len(cmdWords) && strings.Join(toks[:len(cmdWords)], " ") == command:
		toks = toks[len(cmdWords):]
	case len(toks) > 0 && toks[0] == strings.Join(cmdWords, "-"):
		toks = toks[1:]
	default:
		return nil
	}

	var args []Arg
	for _, t := range toks {
		if t == "..." || t == "…" {
			if len(args) == 0 {
				return nil
			}
			args[len(args)-1].Repeat = true
			continue
		}
		a, skip, ok := parseArgToken(t)
		if !ok {
			return nil
		}
		if skip {
			continue
		}
		args = append(args, a)
	}
	if len(args) == 0 || len(args) > maxArgs {
		return nil
	}
	return args
}

// parseArgToken reads one usage-line token. skip is true for option groups
// such as "[OPTION]..." or "[-T]"; ok is false when the token is too
// irregular to read.
func parseArgToken(t string) (a Arg, skip, ok bool) {
	if strings.HasSuffix(t, "...") || strings.HasSuffix(t, "…") {
		a.Repeat = true
		t = strings.TrimRight(strings.TrimSuffix(t, "..."), "…")
	}
	a.Required = true
	if strings.HasPrefix(t, "[") && strings.HasSuffix(t, "]") {
		a.Required = false
		t = t[1 : len(t)-1]
	}
	t = strings.TrimSpace(t)
	// "file ..." inside brackets, "FILE..." inside brackets.
	for _, suf := range []string{" ...", "...", "…"} {
		if strings.HasSuffix(t, suf) {
			a.Repeat = true
			t = strings.TrimSpace(strings.TrimSuffix(t, suf))
		}
	}
	switch {
	case t == "" || t == "--":
		return a, true, true
	case strings.HasPrefix(t, "-") || strings.HasPrefix(t, "+"):
		return a, true, true // an option or option group
	}
	if u := strings.ToUpper(strings.Trim(t, "<>")); u == "OPTION" || u == "OPTIONS" || strings.HasSuffix(u, "-OPTIONS") {
		return a, true, true
	}
	t = strings.Trim(t, "<>")
	// "MODE[,MODE]...": a comma-separated list of one kind of value.
	if i := strings.Index(t, "[,"); i > 0 && t[i:] == "[,"+t[:i]+"]" {
		t, a.Repeat = t[:i], true
	}
	if !argNameRe.MatchString(t) || strings.ContainsAny(t, "[]{}|") {
		return a, false, false
	}
	a.Name = t
	a.Path = argPathRe.MatchString(t)
	return a, false, true
}

// synopsisTokens splits a usage line on spaces outside brackets.
func synopsisTokens(line string) []string {
	var toks []string
	var cur strings.Builder
	depth := 0
	for _, r := range line {
		switch {
		case r == '[' || r == '{' || r == '<' || r == '(':
			depth++
		case (r == ']' || r == '}' || r == '>' || r == ')') && depth > 0:
			depth--
		case unicode.IsSpace(r) && depth == 0:
			if cur.Len() > 0 {
				toks = append(toks, cur.String())
				cur.Reset()
			}
			continue
		}
		cur.WriteRune(r)
	}
	if cur.Len() > 0 {
		toks = append(toks, cur.String())
	}
	return toks
}
