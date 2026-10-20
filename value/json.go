package value

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
)

// jsonIndent is what one level of nesting adds to the indentation.
const jsonIndent = "  "

// jsonClosing pairs each opening delimiter with the one that closes it.
var jsonClosing = map[json.Delim]json.Delim{'{': '}', '[': ']'}

// jsonValue is a JSON document rendered into lines.
type jsonValue struct {
	lines []Line
}

func (v jsonValue) Lines() []Line {
	return v.lines
}

// decodeJSON renders b as indented JSON. It reports false for anything
// that is not exactly one JSON document, trailing junk included.
func decodeJSON(b []byte) (Value, bool) {
	dec := json.NewDecoder(bytes.NewReader(b))
	// Numbers are kept as they were written: read back as float64 a
	// large integer would lose digits and 1.0 would show as 1.
	dec.UseNumber()

	var w jsonWriter
	if err := w.value(dec); err != nil {
		return nil, false
	}
	// The decoder stops at the end of the first document, so anything
	// but trailing space means these bytes are not one JSON value.
	if _, err := dec.Token(); !errors.Is(err, io.EOF) {
		return nil, false
	}
	w.newline()

	return jsonValue{lines: w.lines}, true
}

// jsonWriter turns the token stream of a decoder into lines. It holds
// the line being written until a newline moves it into lines.
type jsonWriter struct {
	lines []Line
	line  Line
	depth int
}

// value renders the next value of the stream, which may be a whole
// object or array.
func (w *jsonWriter) value(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}

	switch token := token.(type) {
	case json.Delim:
		return w.container(dec, token)
	case string:
		w.add(quoteJSON(token), KindString)
	case json.Number:
		w.add(token.String(), KindNumber)
	case bool:
		w.add(strconv.FormatBool(token), KindLiteral)
	case nil:
		w.add("null", KindLiteral)
	default:
		return fmt.Errorf("unexpected token %v of type %T", token, token)
	}

	return nil
}

// container renders the object or array opened by open, one element per
// line, with the closing delimiter back at the opening indentation.
func (w *jsonWriter) container(dec *json.Decoder, open json.Delim) error {
	end, ok := jsonClosing[open]
	if !ok {
		return fmt.Errorf("unexpected delimiter %q", open)
	}
	if !dec.More() {
		// An empty container is not worth two lines.
		w.add(string(open)+string(end), KindPunct)

		return skip(dec)
	}

	w.add(string(open), KindPunct)
	w.newline()
	w.depth++
	for dec.More() {
		w.indent()
		if open == '{' {
			if err := w.key(dec); err != nil {
				return err
			}
		}
		if err := w.value(dec); err != nil {
			return err
		}
		// The last element of a container carries no comma.
		if dec.More() {
			w.add(",", KindPunct)
		}
		w.newline()
	}
	w.depth--
	w.indent()
	w.add(string(end), KindPunct)

	return skip(dec)
}

// key renders the name of the object member that comes next, and the
// colon that separates it from its value.
func (w *jsonWriter) key(dec *json.Decoder) error {
	token, err := dec.Token()
	if err != nil {
		return err
	}
	name, ok := token.(string)
	if !ok {
		return fmt.Errorf("object key %v is a %T, not a string", token, token)
	}
	w.add(quoteJSON(name), KindKey)
	w.add(":", KindPunct)
	w.add(" ", KindPlain)

	return nil
}

// add appends a token to the line being written. Empty tokens are
// dropped, so the indentation of the outermost level leaves no trace.
func (w *jsonWriter) add(text string, kind Kind) {
	if text == "" {
		return
	}
	w.line = append(w.line, Token{Text: text, Kind: kind})
}

// indent opens a line with the whitespace of the current nesting depth.
func (w *jsonWriter) indent() {
	w.add(strings.Repeat(jsonIndent, w.depth), KindPlain)
}

// newline moves the line being written into lines.
func (w *jsonWriter) newline() {
	w.lines = append(w.lines, w.line)
	w.line = nil
}

// skip reads the token that closes a container, which the caller has
// already rendered.
func skip(dec *json.Decoder) error {
	_, err := dec.Token()

	return err
}

// quoteJSON renders s as a JSON string literal. It leaves <, > and & as
// they were written: this text is read, and escaping it for a particular
// interface is that interface's job.
func quoteJSON(s string) string {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(s); err != nil {
		return strconv.Quote(s)
	}

	// Encode ends every document with a newline.
	return strings.TrimSuffix(buf.String(), "\n")
}
