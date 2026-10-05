// Package shellword splits command lines into words the way a POSIX shell
// does, keeping both the typed text and the unquoted value.
package shellword

import "strings"

// Word is a shell word: Raw is the text as typed, Value has quotes removed.
type Word struct {
	Raw   string
	Value string
}

// Split splits a command line into shell words, honoring quotes and
// backslashes. Unterminated quotes run to the end of input.
func Split(line string) []Word {
	var words []Word
	var raw, val strings.Builder
	inWord := false
	quote := byte(0)
	flush := func() {
		if inWord {
			words = append(words, Word{Raw: raw.String(), Value: val.String()})
		}
		raw.Reset()
		val.Reset()
		inWord = false
	}
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quote == '\'':
			raw.WriteByte(c)
			if c == '\'' {
				quote = 0
			} else {
				val.WriteByte(c)
			}
		case quote == '"':
			raw.WriteByte(c)
			if c == '"' {
				quote = 0
			} else if c == '\\' && i+1 < len(line) && strings.IndexByte("\"\\$`", line[i+1]) >= 0 {
				i++
				raw.WriteByte(line[i])
				val.WriteByte(line[i])
			} else {
				val.WriteByte(c)
			}
		case c == ' ' || c == '\t' || c == '\n':
			flush()
		case c == '\'' || c == '"':
			inWord = true
			quote = c
			raw.WriteByte(c)
		case c == '\\' && i+1 < len(line):
			inWord = true
			raw.WriteByte(c)
			i++
			raw.WriteByte(line[i])
			val.WriteByte(line[i])
		default:
			inWord = true
			raw.WriteByte(c)
			val.WriteByte(c)
		}
	}
	flush()
	return words
}
