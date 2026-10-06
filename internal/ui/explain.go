package ui

import (
	"context"
	"regexp"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/lonedevel/commando/internal/cmdline"
	"github.com/lonedevel/commando/internal/manpage"
)

// Explain describes a shell command line: each command in it, with what
// each of its options and arguments means according to its manual. It is
// what commando --explain prints, wrapped to width.
func Explain(ctx context.Context, line string, width int, useCache bool) string {
	width = max(40, width)
	var out []string
	segs := splitLine(line)
	if len(segs) == 0 {
		return sDim.Render("Nothing to explain.") + "\n"
	}
	for i, seg := range segs {
		if i > 0 {
			out = append(out, "")
		}
		if seg.sep != "" {
			out = append(out, sKey.Render(seg.sep)+" "+sDim.Render(separatorText[seg.sep]), "")
		}
		out = append(out, explainCommand(ctx, seg.text, width, useCache)...)
	}
	return strings.Join(out, "\n") + "\n"
}

// separatorText says what joins two commands.
var separatorText = map[string]string{
	"|":  "sends its output into",
	"|&": "sends its output and errors into",
	"&&": "then, if that succeeded,",
	"||": "or, if that failed,",
	";":  "then",
	"&":  "runs in the background; then",
}

// wrappers run the command that follows them.
var wrappers = map[string]string{
	"sudo":    "runs the command after it as another user (root by default)",
	"doas":    "runs the command after it as another user (root by default)",
	"time":    "runs the command after it and reports how long it took",
	"nohup":   "runs the command after it, ignoring hangups, so it keeps running after you log out",
	"nice":    "runs the command after it with a lower scheduling priority",
	"env":     "runs the command after it with the environment changed",
	"xargs":   "runs the command after it with arguments read from its input",
	"command": "runs the command after it, ignoring shell functions and aliases",
	"exec":    "replaces the shell with the command after it",
}

var (
	assignRe   = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*=`)
	redirectRe = regexp.MustCompile(`^(\d*|&)(>>|>\||>&|>|<<<|<<|<&|<)(.*)$`)
)

// explainCommand describes one simple command, and any it wraps.
func explainCommand(ctx context.Context, text string, width int, useCache bool) []string {
	words := cmdline.Split(text)
	var out []string
	// Redirections and NAME=value settings apply to the whole command.
	var notes []string
	words, notes = takeRedirects(words)
	for len(words) > 0 && assignRe.MatchString(words[0].Value) {
		name, val, _ := strings.Cut(words[0].Value, "=")
		out = append(out, explainRow(sArg.Render(words[0].Raw), "sets "+name+" to "+cmdline.Quote(val)+" for this command", sText, width)...)
		words = words[1:]
	}
	if len(words) == 0 {
		return append(out, notes...)
	}
	vals := make([]string, len(words))
	for i, w := range words {
		vals[i] = w.Value
	}
	spec, n, err := manpage.LoadLine(ctx, vals, useCache)
	if err != nil {
		out = append(out, sCmd.Render(vals[0])+sDim.Render(" — no manual or --help found, so this can't be explained"))
		return append(out, notes...)
	}
	rest := words[n:]
	// A wrapper's own options come first; the command it runs is explained
	// on its own.
	var inner []cmdline.Word
	if desc, ok := wrappers[spec.Command]; ok {
		used := wrapperOptions(spec, rest)
		rest, inner = rest[:used], rest[used:]
		if len(inner) > 0 {
			spec = withSummary(spec, desc)
		}
	}
	out = append(out, commandHeader(spec))
	out = append(out, explainParts(spec, cmdline.Explain(spec, rest), width, false)...)
	if len(rest) == 0 && len(inner) == 0 {
		out = append(out, "  "+sDim.Render("no options or arguments"))
	}
	out = append(out, notes...)
	if len(inner) > 0 {
		var raws []string
		for _, w := range inner {
			raws = append(raws, w.Raw)
		}
		out = append(out, "")
		out = append(out, explainCommand(ctx, strings.Join(raws, " "), width, useCache)...)
	}
	return out
}

// wrapperOptions counts the words at the start of words that are the
// wrapper's own options ("-u bob" in sudo -u bob ls), with their values.
func wrapperOptions(spec *manpage.Spec, words []cmdline.Word) int {
	byName := map[string]*manpage.Option{}
	for i := range spec.Options {
		for _, n := range spec.Options[i].Names {
			byName[n] = &spec.Options[i]
		}
	}
	k := 0
	for k < len(words) {
		v := words[k].Value
		if v == "--" {
			return k + 1
		}
		if !strings.HasPrefix(v, "-") || v == "-" {
			break
		}
		k++
		name, _, eq := strings.Cut(v, "=")
		o := byName[name]
		if o == nil && !strings.HasPrefix(v, "--") && len(v) > 2 {
			o = byName[v[len(v)-2:]] // the last of a cluster: -0I
		}
		if o != nil && o.TakesArg() && !o.ArgOptional && !eq && (len(v) == 2 || strings.HasPrefix(v, "--") || byName[v] == nil) && k < len(words) {
			k++ // its value
		}
	}
	return k
}

func withSummary(s *manpage.Spec, summary string) *manpage.Spec {
	c := *s
	c.Summary = summary
	return &c
}

func commandHeader(s *manpage.Spec) string {
	h := sCmd.Render(s.Command)
	if s.Summary != "" {
		h += sDim.Render(" — " + s.Summary)
	}
	return h
}

// takeRedirects removes redirections ("> out.txt", "2>&1") from words and
// describes them.
func takeRedirects(words []cmdline.Word) ([]cmdline.Word, []string) {
	var kept []cmdline.Word
	var notes []string
	for i := 0; i < len(words); i++ {
		w := words[i]
		m := redirectRe.FindStringSubmatch(w.Raw)
		if m == nil {
			kept = append(kept, w)
			continue
		}
		fd, op, target := m[1], m[2], m[3]
		text := w.Raw
		if target == "" && i+1 < len(words) && op != ">&" && op != "<&" {
			i++
			target = words[i].Raw
			text += " " + target
		}
		desc, risky := redirectText(fd, op, target)
		st := sText
		if risky {
			st = sErr
		}
		notes = append(notes, explainRow(sValue.Render(text), desc, st, 100)...)
	}
	return kept, notes
}

// redirectText describes a redirection, and whether it replaces a file.
func redirectText(fd, op, target string) (string, bool) {
	stream := "output"
	switch fd {
	case "2":
		stream = "errors"
	case "&":
		stream = "output and errors"
	}
	switch {
	case op == ">&" || strings.HasPrefix(target, "&"):
		to := map[string]string{"1": "output", "2": "errors"}[strings.TrimPrefix(target, "&")]
		if to == "" {
			to = "that stream"
		}
		return "sends " + stream + " where " + to + " goes", false
	case target == "/dev/null" && (op == ">" || op == ">>"):
		return "discards " + stream, false
	case op == ">" || op == ">|":
		return "writes " + stream + " to " + target + ", replacing what it holds", true
	case op == ">>":
		return "appends " + stream + " to " + target, false
	case op == "<<" || op == "<<<":
		return "reads input from the text that follows", false
	}
	return "reads input from " + target, false
}

// explainParts describes each option and argument. compact puts the text
// and its description on separate lines, for narrow panes.
func explainParts(spec *manpage.Spec, parts []cmdline.Part, width int, compact bool) []string {
	type note struct {
		text  string
		style lipgloss.Style
	}
	var out []string
	for _, p := range parts {
		var tok string
		var notes []note
		switch p.Kind {
		case cmdline.PartOption:
			o := &spec.Options[p.Opt]
			tok = styledPart(p)
			desc := optionLabel(o)
			if other := otherNames(o, p.Text); other != "" {
				desc += " (" + other + ")"
			}
			notes = append(notes, note{desc, sText})
			if len(o.Choices) > 0 && p.Value != "" && indexOf(o.Choices, p.Value) < 0 {
				notes = append(notes, note{"not one of the listed values: " + strings.Join(o.Choices, ", "), sDim})
			}
			if o.Danger != "" {
				notes = append(notes, note{"⚠ " + o.Danger, sErr})
			}
		case cmdline.PartArg:
			tok = sArg.Render(p.Text)
			desc := "argument"
			if p.Arg != "" {
				desc = manpage.Arg{Name: p.Arg}.Label() + " (argument)"
			}
			notes = append(notes, note{desc, sText})
		case cmdline.PartUnknown:
			tok = sNameB.Render(p.Text)
			notes = append(notes, note{"not in the manual for " + spec.Command, sValue})
		case cmdline.PartOperator:
			tok = sKey.Render(p.Text)
			notes = append(notes, note{operatorText[p.Value], sText})
		case cmdline.PartEnd:
			tok = sKey.Render(p.Text)
			notes = append(notes, note{"end of options: what follows are arguments, even if they start with -", sText})
		}
		if compact {
			out = append(out, fit(tok, width))
			for _, n := range notes {
				for _, l := range wrap(n.text, max(10, width-2)) {
					out = append(out, "  "+n.style.Render(l))
				}
			}
			continue
		}
		for i, n := range notes {
			t := tok
			if i > 0 {
				t = ""
			}
			out = append(out, explainRow(t, n.text, n.style, width)...)
		}
	}
	return out
}

var operatorText = map[string]string{
	"!":    "not: the test after it must fail",
	"-not": "not: the test after it must fail",
	"(":    "starts a group of tests",
	")":    "ends a group of tests",
	"-o":   "or: either the test before or the one after",
	"-or":  "or: either the test before or the one after",
	"-a":   "and: both tests (the default between tests)",
	"-and": "and: both tests (the default between tests)",
	",":    "both expressions run; the second one's result counts",
}

// styledPart colors an option as written: the name, then its value.
func styledPart(p cmdline.Part) string {
	name, val, ok := strings.Cut(p.Text, "=")
	if ok && strings.HasPrefix(name, "--") {
		return sNameB.Render(name+"=") + sValue.Render(val)
	}
	if name, val, ok := strings.Cut(p.Text, " "); ok {
		return sNameB.Render(name) + " " + sValue.Render(val)
	}
	return sNameB.Render(p.Text)
}

// optionLabel is the first sentence of the option's description, which
// has room here, unlike the form's short label.
func optionLabel(o *manpage.Option) string {
	d := strings.Join(strings.Fields(firstPara(o.Desc)), " ")
	if i := strings.Index(d, ". "); i > 0 {
		d = d[:i+1]
	}
	switch {
	case d != "" && len(d) <= 200:
		return d
	case o.Label != "":
		return o.Label
	}
	return "(no description)"
}

// otherNames lists the option's other spellings: "-v" for --verbose.
func otherNames(o *manpage.Option, written string) string {
	used, _, _ := strings.Cut(written, "=")
	used, _, _ = strings.Cut(used, " ")
	var names []string
	for _, n := range o.Names {
		if n != used && !strings.HasPrefix(used, n) {
			names = append(names, n)
		}
	}
	return strings.Join(names, ", ")
}

// explainRow lays out the text as written beside its description,
// wrapping the description under itself.
func explainRow(tok, desc string, st lipgloss.Style, width int) []string {
	const col = 26
	tw := ansi.StringWidth(tok)
	var out []string
	first := "  " + tok + strings.Repeat(" ", max(1, col-tw))
	indent := "  " + strings.Repeat(" ", col)
	if tw > col-1 {
		out = append(out, "  "+tok)
		first = indent
	}
	for i, l := range wrap(desc, max(20, width-col-2)) {
		if i == 0 {
			out = append(out, first+st.Render(l))
		} else {
			out = append(out, indent+st.Render(l))
		}
	}
	return out
}

type segment struct {
	text string
	sep  string // what joins it to the command before: "|", "&&"…
}

// splitLine splits a shell line into commands at |, &&, ||, ; and &,
// outside quotes.
func splitLine(line string) []segment {
	var segs []segment
	var b strings.Builder
	sep := ""
	var quote byte
	flush := func(next string) {
		if t := strings.TrimSpace(b.String()); t != "" {
			segs = append(segs, segment{text: t, sep: sep})
		}
		b.Reset()
		sep = next
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
			} else if c == '\\' && quote == '"' && i+1 < len(line) {
				b.WriteByte(c)
				i++
				c = line[i]
			}
		case c == '\\' && i+1 < len(line):
			b.WriteByte(c)
			i++
			c = line[i]
		case c == '\'' || c == '"':
			quote = c
		case c == '|':
			if i+1 < len(line) && (line[i+1] == '|' || line[i+1] == '&') {
				flush(line[i : i+2])
				i++
			} else if i > 0 && line[i-1] == '>' {
				b.WriteByte(c) // >| clobber
				continue
			} else {
				flush("|")
			}
			continue
		case c == ';':
			flush(";")
			continue
		case c == '&':
			prevRedir := i > 0 && (line[i-1] == '>' || line[i-1] == '<')
			nextRedir := i+1 < len(line) && line[i+1] == '>'
			if prevRedir || nextRedir {
				break // 2>&1, &> file
			}
			if i+1 < len(line) && line[i+1] == '&' {
				flush("&&")
				i++
			} else {
				flush("&")
			}
			continue
		}
		b.WriteByte(c)
	}
	flush("")
	return segs
}
