package ui

import (
	"bytes"
	"encoding/json"
	"strings"
	"unicode"
)

// jsonIndent is the indentation of a rendered value.
const jsonIndent = "  "

// jsonPunctuation are the structural characters of JSON.
const jsonPunctuation = "{}[],:"

// formatValue renders a stored value as indented JSON, split into lines.
// Values that are not valid JSON yield no lines at all: for now the UI
// only serves JSON and shows nothing for anything else.
func formatValue(value []byte) []string {
	var buf bytes.Buffer
	if err := json.Indent(&buf, value, "", jsonIndent); err != nil {
		return nil
	}

	return strings.Split(buf.String(), "\n")
}

// highlightJSON colors one line of rendered JSON: object keys, strings,
// numbers, the true/false/null literals and the braces, brackets and
// separators each get their own style.
//
// It works line by line, on text that may already have been truncated to
// the pane width, so it treats an unterminated token as running to the
// end of the line rather than as an error.
func (s styles) highlightJSON(line string) string {
	var out strings.Builder
	runes := []rune(line)

	for i := 0; i < len(runes); {
		switch r := runes[i]; {
		case r == '"':
			end := scanString(runes, i)
			style := s.jsonString
			if isKey(runes, end) {
				style = s.jsonKey
			}
			out.WriteString(style.Render(string(runes[i:end])))
			i = end

		case r == '-' || unicode.IsDigit(r):
			end := scanWhile(runes, i+1, func(r rune) bool {
				return unicode.IsDigit(r) || strings.ContainsRune(".eE+-", r)
			})
			out.WriteString(s.jsonNumber.Render(string(runes[i:end])))
			i = end

		case unicode.IsLetter(r):
			end := scanWhile(runes, i, unicode.IsLetter)
			out.WriteString(s.jsonLiteral.Render(string(runes[i:end])))
			i = end

		case strings.ContainsRune(jsonPunctuation, r):
			out.WriteString(s.jsonPunct.Render(string(r)))
			i++

		default:
			out.WriteRune(r)
			i++
		}
	}

	return out.String()
}

// scanString returns the index just past the string literal that starts
// at the quote at position start, or the end of the line if the quote is
// never closed.
func scanString(runes []rune, start int) int {
	for i := start + 1; i < len(runes); i++ {
		switch runes[i] {
		case '\\':
			i++ // Skip whatever the backslash escapes, quotes included.
		case '"':
			return i + 1
		}
	}

	return len(runes)
}

// scanWhile returns the index of the first rune at or after start that
// does not satisfy keep.
func scanWhile(runes []rune, start int, keep func(rune) bool) int {
	for i := start; i < len(runes); i++ {
		if !keep(runes[i]) {
			return i
		}
	}

	return len(runes)
}

// isKey reports whether the string that ends at index end is an object
// key, i.e. whether a colon follows it.
func isKey(runes []rune, end int) bool {
	for i := end; i < len(runes); i++ {
		if runes[i] == ' ' {
			continue
		}

		return runes[i] == ':'
	}

	return false
}
