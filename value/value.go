// Package value decodes stored values into lines ready to be displayed.
//
// A bbolt value is an untyped blob, so the format is a guess made from
// the bytes themselves. Decode makes that guess and hands back a Value,
// which renders the blob as lines of classified tokens. The rendering
// carries no styling of its own: it says what each token is, and leaves
// it to the user interface to decide how a key or a number looks.
package value

// A Kind classifies a token, so an interface can style it.
type Kind int

const (
	// KindPlain is text that carries no meaning of its own, such as
	// the indentation at the start of a line.
	KindPlain Kind = iota
	KindKey
	KindString
	KindNumber
	// KindLiteral is a keyword of the format, such as JSON's true,
	// false and null.
	KindLiteral
	// KindPunct is structural, such as JSON's braces and separators.
	KindPunct
)

// A Token is a run of text of a single kind.
type Token struct {
	Text string
	Kind Kind
}

// A Line is one line of a rendered value, split into tokens. Lines carry
// no trailing newline.
type Line []Token

// Text returns the line as plain text, dropping the token boundaries.
func (l Line) Text() string {
	text := ""
	for _, token := range l {
		text += token.Text
	}

	return text
}

// A Value is a stored value decoded into displayable lines.
type Value interface {
	// Lines returns the value rendered for display, one Line per row.
	Lines() []Line
}

// A decoder renders bytes in the one format it knows, and reports false
// for bytes in any other.
type decoder func([]byte) (Value, bool)

// decoders are tried in order, so the stricter formats come first.
var decoders = []decoder{decodeJSON}

// Decode renders b in the first format that recognizes it. It reports
// false if no format does, leaving it to the caller to decide what to
// show for a blob it cannot read.
func Decode(b []byte) (Value, bool) {
	for _, decode := range decoders {
		if value, ok := decode(b); ok {
			return value, true
		}
	}

	return nil, false
}
