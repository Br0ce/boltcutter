package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// binding is a single shortcut: the keys that trigger it and how it is
// labelled in the footer.
type binding struct {
	keys  []string
	label string // the key spelling shown to the user.
	help  string
}

var (
	keyQuit     = binding{keys: []string{"q", "ctrl+c"}, label: "q", help: "quit"}
	keyFocus    = binding{keys: []string{"tab"}, label: "tab", help: "focus value"}
	keyUp       = binding{keys: []string{"up", "k"}, label: "↑/k", help: "up"}
	keyDown     = binding{keys: []string{"down", "j"}, label: "↓/j", help: "down"}
	keyPageUp   = binding{keys: []string{"pgup", "ctrl+b"}, label: "pgup", help: "page up"}
	keyPageDown = binding{keys: []string{"pgdown", "ctrl+f"}, label: "pgdn", help: "page down"}
	keyTop      = binding{keys: []string{"home", "g"}, label: "g", help: "top"}
	keyBottom   = binding{keys: []string{"end", "G"}, label: "G", help: "bottom"}
	keyOpen     = binding{keys: []string{"enter", "right", "l"}, label: "enter/→", help: "open bucket"}
	keyBack     = binding{keys: []string{"esc", "backspace", "left", "h"}, label: "esc/←", help: "back"}
)

// footerBindings lists the shortcuts of the footer, in display order.
var footerBindings = []binding{
	keyUp, keyDown, keyOpen, keyBack, keyFocus, keyQuit,
}

// keyMatches reports whether msg triggers b.
func keyMatches(msg tea.KeyMsg, b binding) bool {
	pressed := msg.String()
	for _, key := range b.keys {
		if key == pressed {
			return true
		}
	}

	return false
}

// footer renders the shortcut bar spanning the full width. It drops
// bindings from the right when the terminal is too narrow to hold them
// all, rather than wrapping into a second line.
func (m Model) footer() string {
	var (
		parts []string
		width int
	)
	const separator = "  "
	for _, b := range footerBindings {
		part := m.styles.footerKey.Render(b.label) + " " + b.help
		next := width + len(b.label) + 1 + len(b.help)
		if len(parts) > 0 {
			next += len(separator)
		}
		// The footer style adds a border and a column of padding on
		// each side.
		if next > m.width-boxBorder-boxPadding {
			break
		}
		parts = append(parts, part)
		width = next
	}

	return m.styles.footer.Width(max(m.width-boxBorder, 0)).Render(strings.Join(parts, separator))
}
