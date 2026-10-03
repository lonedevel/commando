package manpage

import (
	"regexp"
	"strings"
)

var (
	introRe     = regexp.MustCompile(`(?i)(?:\bpossible values?:|\bvalid values?:|\ballowed values?:|\bcan (?:also )?be(?: one of)?|\bmay (?:also )?be(?: set to| one of)?|\bmust be(?: one of)?|\bis one of|\bone of|\bvalues? (?:are|is|include)|\barguments? (?:are|is)|\bchoices are|\bsupported (?:values|types|formats) are|:)\s+`)
	defaultRe   = regexp.MustCompile(`(?i)\bdefaults? (?:to|is) ['‘"“]?([A-Za-z0-9][\w.+-]*)`)
	choiceTokRe = regexp.MustCompile(`^[A-Za-z0-9][\w.+-]*$`)
	wordFlagRe  = regexp.MustCompile(`^([a-z][\w-]*) \(?-[A-Za-z0-9]\)?$`)
	quotes      = "'‘’\"“”`"
)

var stopWords = map[string]bool{
	"the": true, "a": true, "an": true, "it": true, "this": true, "that": true,
	"used": true, "specified": true, "given": true, "set": true, "be": true,
	"is": true, "are": true, "not": true, "also": true, "e.g": true, "i.e": true, "eg": true, "ie": true, "see": true,
	"default": true, "etc": true, "and": true, "or": true, "if": true, "only": true,
	"in": true, "to": true, "of": true, "for": true, "with": true, "any": true,
	"none.": true, "same": true, "one": true,
}

func cleanItem(s string) (string, bool) {
	s = strings.TrimSpace(s)
	for _, pre := range []string{"either ", "or ", "and "} {
		s = strings.TrimPrefix(s, pre)
	}
	s = strings.Trim(s, quotes+" []")
	s = strings.TrimRight(s, ".,:;]")
	s = strings.Trim(s, quotes)
	if s == "" || len(s) > 24 || !choiceTokRe.MatchString(s) || stopWords[strings.ToLower(s)] {
		return "", false
	}
	return s, true
}

var listSplitRe = regexp.MustCompile(`\s*,\s*(?:or\s+|and\s+)?|\s+or\s+|\s+and\s+|\s*\|\s*`)

var exampleRe = regexp.MustCompile(`(?i)(example|e\.g|such as|like|for instance|supported by|built with)`)

// parseList reads "always, auto, or never" from the start of s.
func parseList(s string) []string {
	s = parenRe.ReplaceAllString(s, "")
	var out []string
	for _, part := range listSplitRe.Split(s, -1) {
		it, ok := cleanItem(part)
		if !ok {
			break
		}
		out = append(out, it)
	}
	if len(out) < 2 {
		return nil
	}
	for _, it := range out {
		if it[0] >= '0' && it[0] <= '9' {
			return nil // sample values ("200K, 3m and 1G"), not choices
		}
	}
	return out
}

// choicesFor derives an option's allowed values from its documentation.
func (p *parser) choicesFor(o *Option) []string {
	var out []string
	seen := map[string]bool{}
	add := func(items ...string) {
		for _, it := range items {
			if it == "" || seen[it] || strings.EqualFold(it, o.Arg) {
				continue
			}
			seen[it] = true
			out = append(out, it)
		}
	}

	// {a,b,c} placeholders, common in --help output, and a|b|c in
	// clap-style help ("--edition 2015|2018|2021").
	if strings.HasPrefix(o.Arg, "{") || (p.spec.Source == "help" && strings.Contains(o.Arg, "|") && !strings.ContainsAny(o.Arg, "<[=")) {
		a := strings.Trim(o.Arg, "{}()[]<>")
		for _, it := range strings.FieldsFunc(a, func(r rune) bool { return r == ',' || r == '|' }) {
			if c, ok := cleanItem(it); ok {
				add(c)
			}
		}
		if len(out) >= 2 {
			return out
		}
	}

	argRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(strings.Trim(o.Arg, "<>[]")) + `\b`)
	var ifs []string
	// lead is true for text where the option's values are likely described:
	// the opening paragraphs of its description.
	scan := func(text string, lead bool) {
		for _, sent := range sentences(text) {
			for _, m := range introRe.FindAllStringIndex(sent, -1) {
				prefix := sent[:m[0]]
				if exampleRe.MatchString(prefix) {
					continue
				}
				if !lead && !argRe.MatchString(sent) {
					continue
				}
				// A bare colon only introduces values when the clause talks
				// about the argument itself ("sort by WORD instead of name:").
				if strings.TrimSpace(sent[m[0]:m[1]]) == ":" && !argRe.MatchString(prefix) &&
					!(lead && isUpperWord(o.Arg) && strings.Contains(firstParas(o.Desc, 2), o.Arg)) {
					continue
				}
				add(parseList(sent[m[1]:])...)
			}
			for _, m := range defaultRe.FindAllStringSubmatch(sent, -1) {
				if len(out) > 0 || argRe.MatchString(sent) {
					if c, ok := cleanItem(m[1]); ok {
						add(c)
					}
				}
			}
		}
		// "If TYPE is binary, ...", "By default, ACTION is read",
		// "WHEN is never, always, or auto".
		isRe := regexp.MustCompile(`\b` + regexp.QuoteMeta(o.Arg) + `\s+is\s+`)
		for _, m := range isRe.FindAllStringIndex(text, -1) {
			rest := text[m[1]:]
			if l := parseList(rest); l != nil {
				add(l...)
				continue
			}
			if f := strings.Fields(rest); len(f) > 0 && strings.ContainsAny(f[0][len(f[0])-1:], ",.;") {
				if c, ok := cleanItem(f[0]); ok {
					ifs = append(ifs, c)
				}
			}
		}
	}

	for i, pa := range strings.Split(o.Desc, "\n\n") {
		scan(pa, i < 2)
	}
	if len(dedupe(ifs)) >= 2 || (len(ifs) > 0 && len(out) > 0) {
		add(ifs...)
	}

	// "across -x, commas -m, long -l" (GNU ls --format).
	if len(out) < 2 {
		var words []string
		for _, part := range strings.Split(firstParas(o.Desc, 1), ",") {
			m := wordFlagRe.FindStringSubmatch(strings.TrimSpace(spacesRe.ReplaceAllString(part, " ")))
			if m == nil {
				words = nil
				break
			}
			words = append(words, m[1])
		}
		if len(words) >= 2 {
			add(words...)
		}
	}

	// "The WHEN argument defaults to 'always' and can also be 'auto' or
	// 'never'." — general paragraphs about a distinctive placeholder.
	if isUpperWord(o.Arg) && len(o.Arg) >= 3 && !genericArg[o.Arg] {
		for _, pa := range p.paras {
			if argRe.MatchString(pa.text) {
				for _, sent := range sentences(pa.text) {
					if argRe.MatchString(sent) {
						scan(sent, true)
					}
				}
			}
		}
	}

	// Values used in examples elsewhere: "--color=never", "--color=auto".
	if len(out) >= 1 || o.ArgOptional {
		for _, long := range o.Long() {
			re := regexp.MustCompile(regexp.QuoteMeta(long) + `=([a-z][\w-]*)\b`)
			var ex []string
			for _, m := range re.FindAllStringSubmatch(p.spec.Manual, -1) {
				if c, ok := cleanItem(m[1]); ok {
					ex = append(ex, c)
				}
			}
			if len(out) >= 2 || len(dedupe(ex)) >= 2 {
				add(ex...)
			}
		}
	}
	return out
}

var genericArg = map[string]bool{
	"FILE": true, "DIR": true, "PATTERN": true, "PATTERNS": true, "STRING": true,
	"NAME": true, "PATH": true, "TEXT": true, "ARG": true, "VALUE": true,
	"COMMAND": true, "LIST": true, "GLOB": true, "NUM": true, "SIZE": true,
}

func dedupe(in []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range in {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}
