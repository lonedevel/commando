package manpage

import (
	"regexp"
	"strings"
)

// Value lists: what each of an option's values means. Manuals give them as
// a list under the option ("d   directory"), as sentences ("If TYPE is
// text, …"), one paragraph per value ("--date=relative shows …"), or
// inline, naming the equivalent flag ("none (-U), size (-S)").

var (
	// "b      block (buffered) special", "pax, posix   POSIX format",
	// "binary (default)" on a line of its own.
	defRowRe   = regexp.MustCompile(`^([A-Za-z0-9][\w.+-]*(?:,\s*[A-Za-z0-9][\w.+-]*)*)(?:\s+\(default\))?(?:\s{2,}(\S.*))?$`)
	ifIsRe     = regexp.MustCompile(`(?:^|[.;]\s+)(?:If|When) (\S+) is ['‘"“]?([A-Za-z0-9][\w.+-]*)['’"”]?, ([^.]*[^.\s])`)
	wordFlagIn = regexp.MustCompile(`\b([a-z][\w-]*) \((-[A-Za-z0-9])\)`)
	groupRe    = regexp.MustCompile(`([A-Za-z][^;:]*?)(?:\s*\((-[A-Za-z0-9]|default)\))?:\s*([a-z][\w-]*(?:,\s*[a-z][\w-]*)+)\s*(?:;|$)`)
)

// definitionList reads "value  meaning" rows from an option's own lines,
// where a list of values is introduced by a line ending in ":". intro is
// that line, for the first list.
func definitionList(block []line) (values []string, descs map[string]string, intro string) {
	descs = map[string]string{}
	type row struct {
		names []string
		desc  string
	}
	prev := ""
	for i := 0; i < len(block); i++ {
		l := block[i]
		if l.blank() {
			continue
		}
		if !strings.HasSuffix(prev, ":") {
			prev = l.s
			continue
		}
		// A list may start here: rows at this line's indent.
		var rows []row
		descCol := 0 // where the meanings start, once a row shows it
		j := i
		for j < len(block) {
			r := block[j]
			if r.blank() {
				j++
				continue
			}
			if r.ind != l.ind {
				break
			}
			m := defRowRe.FindStringSubmatch(r.s)
			if m == nil && descCol > 0 && descCol < len(r.raw) && r.raw[descCol-1] == ' ' && r.raw[descCol] != ' ' {
				// A value as wide as its column: one space before the
				// meaning, which starts where the other rows' do
				// ("oldgnu GNU format").
				m = defRowRe.FindStringSubmatch(strings.TrimSpace(r.raw[:descCol]))
				if m != nil && m[2] == "" {
					m[2] = strings.TrimSpace(r.raw[descCol:])
				}
			}
			if m == nil || strings.HasSuffix(m[1], ".") {
				break
			}
			if m[2] != "" && descCol == 0 {
				descCol = strings.Index(r.raw, m[2])
			}
			desc := m[2]
			// The meaning may continue, or start, on deeper lines below.
			k := j + 1
			for k < len(block) && (block[k].blank() || block[k].ind > l.ind) {
				if !block[k].blank() {
					desc += " " + block[k].s
				}
				k++
			}
			if strings.TrimSpace(desc) == "" {
				break
			}
			var names []string
			for _, n := range strings.Split(m[1], ",") {
				names = append(names, strings.TrimSpace(n))
			}
			rows = append(rows, row{names, cleanValueDesc(desc)})
			j = k
		}
		if len(rows) >= 2 {
			if intro == "" {
				intro = prev
			}
			for _, r := range rows {
				for _, n := range r.names {
					if _, ok := descs[n]; !ok {
						values = append(values, n)
					}
					descs[n] = r.desc
				}
			}
			i = j - 1
		}
		prev = l.s
	}
	if len(values) < 2 {
		return nil, nil, ""
	}
	return values, descs, intro
}

var (
	// Lines that introduce the values an option takes…
	valuesIntroRe = regexp.MustCompile(`(?i)\b(?:possible|valid|allowed|supported|available|one of|types?|values?|modes?|formats?|methods?|styles?|options are|alternatives|keywords?|following)\b`)
	// …and lists of other things: unit suffixes, variables, escapes.
	notValuesRe = regexp.MustCompile(`(?i)\b(?:units?|suffix(?:es)?|variables?|escapes?|directives?|sequences?|placeholders?|environment)\b`)
	envNameRe   = regexp.MustCompile(`^[A-Z][A-Z0-9]*_[A-Z0-9_]+$`)
)

// valueList reports whether a list under an option gives its values: most
// of them are values already known, or the line introducing the list says
// so, and they aren't environment variable names.
func valueList(o *Option, vals []string, intro string) bool {
	known := 0
	for _, v := range vals {
		if indexOfString(o.Choices, v) >= 0 {
			known++
		}
		if envNameRe.MatchString(v) {
			return false
		}
	}
	if known*2 >= len(vals) {
		return true
	}
	return valuesIntroRe.MatchString(intro) && !notValuesRe.MatchString(intro)
}

func cleanValueDesc(s string) string {
	s = strings.Join(strings.Fields(s), " ")
	return strings.TrimSuffix(capitalize(s), ".")
}

// valueDescs fills in o.ChoiceDesc from its description, and adds values
// that only a list under the option gives. byName finds other options, for
// "none (-U)" style lists.
func (p *parser) valueDescs(o *Option, block []line, byName map[string]int) {
	descs := map[string]string{}
	set := func(v, d string) {
		if _, ok := descs[v]; !ok && d != "" {
			descs[v] = d
		}
	}
	// A list under the option.
	if vals, ds, intro := definitionList(block); len(vals) >= 2 && valueList(o, vals, intro) {
		for _, v := range vals {
			set(v, ds[v])
		}
		if o.Kind == KindString || o.Kind == KindChoice {
			for _, v := range vals {
				if indexOfString(o.Choices, v) < 0 {
					o.Choices = append(o.Choices, v)
				}
			}
			o.Kind = KindChoice
		}
	}
	text := strings.Join(strings.Fields(o.Desc), " ")
	// "--date=relative shows dates relative to the current time" — one
	// paragraph per value.
	for _, long := range o.Long() {
		re := regexp.MustCompile(`^` + regexp.QuoteMeta(long) + `=([a-z][\w-]*)(?: \(or ` + regexp.QuoteMeta(long) + `=([a-z][\w-]*)\))? (.+)`)
		var found []string
		for _, pa := range strings.Split(o.Desc, "\n\n") {
			pa = strings.Join(strings.Fields(pa), " ")
			if m := re.FindStringSubmatch(pa); m != nil {
				d := firstSentence(m[3])
				set(m[1], d)
				found = append(found, m[1])
				if m[2] != "" {
					set(m[2], d)
					found = append(found, m[2])
				}
			}
		}
		if len(found) >= 2 && (o.Kind == KindString || o.Kind == KindChoice) {
			for _, v := range found {
				if indexOfString(o.Choices, v) < 0 {
					o.Choices = append(o.Choices, v)
				}
			}
			o.Kind = KindChoice
		}
	}
	if o.Kind != KindChoice {
		return
	}
	// "If TYPE is text, grep processes a binary file as if it were text".
	for _, m := range ifIsRe.FindAllStringSubmatch(text, -1) {
		if strings.EqualFold(m[1], o.Arg) || strings.EqualFold(m[1], "type") {
			set(m[2], capitalize(m[3]))
		}
	}
	// "access time (-u): atime, access, use; metadata change time (-c): …"
	for _, m := range groupRe.FindAllStringSubmatch(text, -1) {
		label := strings.TrimSpace(m[1])
		if i := strings.LastIndexAny(label, ";,"); i >= 0 {
			label = strings.TrimSpace(label[i+1:])
		}
		if m[2] != "" && m[2] != "default" {
			label += " (like " + m[2] + ")"
		}
		for _, v := range strings.Split(m[3], ",") {
			if v = strings.TrimSpace(v); indexOfString(o.Choices, v) >= 0 {
				set(v, capitalize(label))
			}
		}
	}
	// "none (-U), size (-S)": the same as that flag.
	for _, m := range wordFlagIn.FindAllStringSubmatch(text, -1) {
		if indexOfString(o.Choices, m[1]) < 0 {
			continue
		}
		if i, ok := byName[m[2]]; ok && p.spec.Options[i].Label != "" {
			set(m[1], "Like "+m[2]+": "+strings.TrimSuffix(p.spec.Options[i].Label, "."))
		} else {
			set(m[1], "Like "+m[2])
		}
	}
	for v := range descs {
		if indexOfString(o.Choices, v) < 0 {
			delete(descs, v)
		}
	}
	if len(descs) > 0 {
		o.ChoiceDesc = descs
	}
}

func firstSentence(s string) string {
	if i := strings.Index(s, ". "); i > 0 {
		s = s[:i]
	}
	return strings.TrimSuffix(capitalize(strings.TrimSpace(s)), ".")
}
