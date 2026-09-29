package value

import (
	"slices"
	"strings"
	"testing"
)

func TestDecodeRendersJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
		want  []string
	}{
		{
			name:  "object is indented",
			value: `{"id":1,"ok":true}`,
			want:  []string{"{", `  "id": 1,`, `  "ok": true`, "}"},
		},
		{
			name:  "array is indented",
			value: `[1,2]`,
			want:  []string{"[", "  1,", "  2", "]"},
		},
		{
			name:  "nesting indents further",
			value: `{"a":{"b":[null]}}`,
			want:  []string{"{", `  "a": {`, `    "b": [`, "      null", "    ]", "  }", "}"},
		},
		// Empty containers are not worth a line of their own.
		{name: "empty object", value: `{}`, want: []string{"{}"}},
		{name: "empty array", value: `[]`, want: []string{"[]"}},
		{name: "nested empty", value: `{"a":[]}`, want: []string{"{", `  "a": []`, "}"}},
		{name: "scalar", value: `"hello"`, want: []string{`"hello"`}},
		// Numbers keep the spelling they were stored with, which a
		// round trip through float64 would lose.
		{name: "big integer", value: `[12345678901234567890]`, want: []string{"[", "  12345678901234567890", "]"}},
		{name: "trailing zero", value: `[1.0]`, want: []string{"[", "  1.0", "]"}},
		{name: "exponent", value: `[-1.5e3]`, want: []string{"[", "  -1.5e3", "]"}},
		// Strings are quoted the way JSON quotes them, and the
		// characters an HTML page would care about are left alone.
		{name: "escapes", value: `["a \" b"]`, want: []string{"[", `  "a \" b"`, "]"}},
		{name: "html", value: `["a <b> & c"]`, want: []string{"[", `  "a <b> & c"`, "]"}},
		{name: "whitespace is normalized", value: "{\n\t\"id\" : 1 }", want: []string{"{", `  "id": 1`, "}"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			value, ok := Decode([]byte(test.value))
			if !ok {
				t.Fatalf("Decode(%q) = _, false, want a value", test.value)
			}
			if got := texts(value.Lines()); !slices.Equal(got, test.want) {
				t.Errorf("Decode(%q) renders %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestDecodeRejects(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value string
	}{
		{name: "plain text", value: "hello"},
		{name: "truncated", value: `{"id":`},
		{name: "empty", value: ""},
		{name: "binary", value: "\x00\x01A"},
		// A decoder stops after the first document, so trailing junk
		// has to be caught on purpose.
		{name: "trailing junk", value: `{"id":1} nope`},
		{name: "two documents", value: `{"id":1}{"id":2}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if value, ok := Decode([]byte(test.value)); ok {
				t.Errorf("Decode(%q) = %q, true, want false", test.value, texts(value.Lines()))
			}
		})
	}
}

func TestDecodeClassifiesTokens(t *testing.T) {
	t.Parallel()

	value, ok := Decode([]byte(`{"id":1,"name":"jo","ok":null}`))
	if !ok {
		t.Fatal("Decode = _, false, want a value")
	}

	// Every token of the rendered document, in order, so a key is
	// told apart from the string that is its value.
	want := []Token{
		{Text: "{", Kind: KindPunct},
		{Text: "  ", Kind: KindPlain},
		{Text: `"id"`, Kind: KindKey},
		{Text: ":", Kind: KindPunct},
		{Text: " ", Kind: KindPlain},
		{Text: "1", Kind: KindNumber},
		{Text: ",", Kind: KindPunct},
		{Text: "  ", Kind: KindPlain},
		{Text: `"name"`, Kind: KindKey},
		{Text: ":", Kind: KindPunct},
		{Text: " ", Kind: KindPlain},
		{Text: `"jo"`, Kind: KindString},
		{Text: ",", Kind: KindPunct},
		{Text: "  ", Kind: KindPlain},
		{Text: `"ok"`, Kind: KindKey},
		{Text: ":", Kind: KindPunct},
		{Text: " ", Kind: KindPlain},
		{Text: "null", Kind: KindLiteral},
		{Text: "}", Kind: KindPunct},
	}

	var got []Token
	for _, line := range value.Lines() {
		got = append(got, line...)
	}
	if !slices.Equal(got, want) {
		t.Errorf("tokens = %v, want %v", got, want)
	}
}

func TestDecodeNestsDeeply(t *testing.T) {
	t.Parallel()

	// Nesting is rendered by recursion, so a document deep enough to
	// exhaust the decoder must be refused rather than crash.
	deep := strings.Repeat("[", 100000) + strings.Repeat("]", 100000)
	if _, ok := Decode([]byte(deep)); ok {
		t.Error("Decode of a 100000 level deep document = _, true, want false")
	}
}

// texts renders lines as plain text, one string per line.
func texts(lines []Line) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, line.Text())
	}

	return out
}
