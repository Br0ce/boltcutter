package ui

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFormatValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value []byte
		want  []string
	}{
		{
			name:  "object is indented",
			value: []byte(`{"id":1,"ok":true}`),
			want:  []string{"{", `  "id": 1,`, `  "ok": true`, "}"},
		},
		{
			name:  "array is indented",
			value: []byte(`[1,2]`),
			want:  []string{"[", "  1,", "  2", "]"},
		},
		{name: "scalar", value: []byte(`"hello"`), want: []string{`"hello"`}},
		// Anything that is not JSON shows as an empty pane for now.
		{name: "plain text", value: []byte("hello"), want: nil},
		{name: "truncated json", value: []byte(`{"id":`), want: nil},
		{name: "binary", value: []byte{0x00, 0x01, 0x41}, want: nil},
		{name: "empty", value: nil, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := formatValue(test.value)
			if !slices.Equal(got, test.want) {
				t.Errorf("formatValue(%q) = %q, want %q", test.value, got, test.want)
			}
		})
	}
}

func TestHighlightJSON(t *testing.T) {
	t.Parallel()

	// Highlighting only wraps tokens in escape sequences, so stripping
	// the styles must give the line back unchanged.
	tests := []string{
		"{",
		`  "name": "John Doe 1",`,
		`  "id": -1.5e3,`,
		`  "ok": true,`,
		`  "quoted": "a \" b: c",`,
		"  [1, null]",
		"}",
		// Truncation can cut a token in half; that must still render.
		`  "unterminate…`,
	}

	styles := newStyles()
	for _, line := range tests {
		t.Run(line, func(t *testing.T) {
			t.Parallel()

			got := styles.highlightJSON(line)
			if stripped := ansi.Strip(got); stripped != line {
				t.Errorf("highlightJSON(%q) renders %q, want the line unchanged", line, stripped)
			}
		})
	}
}

func TestHighlightJSONStylesTokens(t *testing.T) {
	t.Parallel()

	styles := newStyles()
	line := `  "id": 1,`
	got := styles.highlightJSON(line)

	// A key and a number are styled differently, so the same text must
	// not come out the same way.
	if !strings.Contains(got, styles.jsonKey.Render(`"id"`)) {
		t.Errorf("highlightJSON(%q) = %q, does not style the key", line, got)
	}
	if !strings.Contains(got, styles.jsonNumber.Render("1")) {
		t.Errorf("highlightJSON(%q) = %q, does not style the number", line, got)
	}
}
