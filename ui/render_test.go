package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/boltcutter/value"
)

// line is the rendering of `  "id": 1,`, the shape every test here
// works on.
var line = value.Line{
	{Text: "  ", Kind: value.KindPlain},
	{Text: `"id"`, Kind: value.KindKey},
	{Text: ":", Kind: value.KindPunct},
	{Text: " ", Kind: value.KindPlain},
	{Text: "1", Kind: value.KindNumber},
	{Text: ",", Kind: value.KindPunct},
}

func TestRenderLine(t *testing.T) {
	t.Parallel()

	// Styling only wraps tokens in escape sequences, so stripping the
	// styles must give the line back unchanged.
	got := newStyles().renderLine(line, 40)
	if stripped := ansi.Strip(got); stripped != line.Text() {
		t.Errorf("renderLine renders %q, want %q", stripped, line.Text())
	}
}

func TestRenderLineStylesTokens(t *testing.T) {
	t.Parallel()

	styles := newStyles()
	got := styles.renderLine(line, 40)

	// A key and a number are styled differently, so the same text must
	// not come out the same way.
	if !strings.Contains(got, styles.tokenKey.Render(`"id"`)) {
		t.Errorf("renderLine = %q, does not style the key", got)
	}
	if !strings.Contains(got, styles.tokenNumber.Render("1")) {
		t.Errorf("renderLine = %q, does not style the number", got)
	}
}

func TestRenderLineTruncates(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		width int
		want  string
	}{
		{name: "room to spare", width: 40, want: `  "id": 1,`},
		{name: "exact fit", width: 10, want: `  "id": 1,`},
		// The cut falls inside a token, which keeps its style and
		// carries the ellipsis.
		{name: "cuts a token", width: 5, want: `  "i…`},
		// The cut falls on a token boundary, so the token that no
		// longer fits is dropped whole.
		{name: "cuts at a boundary", width: 6, want: `  "id…`},
		{name: "no room at all", width: 0, want: ""},
		{name: "negative width", width: -1, want: ""},
	}

	styles := newStyles()
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			got := ansi.Strip(styles.renderLine(line, test.width))
			if got != test.want {
				t.Errorf("renderLine(width %d) = %q, want %q", test.width, got, test.want)
			}
			if width := ansi.StringWidth(got); width > max(test.width, 0) {
				t.Errorf("renderLine(width %d) is %d cells wide", test.width, width)
			}
		})
	}
}
