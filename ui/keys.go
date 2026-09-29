package ui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// binding is a single shortcut: the keys that trigger it. What the
// header says about them is kept apart in shortcuts, because the two
// do not line up one to one: a reader is told about moving once,
// while up and down are bound separately.
type binding struct {
	keys []string
}

var (
	keyQuit     = binding{keys: []string{"q", "ctrl+c"}}
	keyFocus    = binding{keys: []string{"tab"}}
	keyUp       = binding{keys: []string{"up", "k"}}
	keyDown     = binding{keys: []string{"down", "j"}}
	keyPageUp   = binding{keys: []string{"pgup", "ctrl+b"}}
	keyPageDown = binding{keys: []string{"pgdown", "ctrl+f"}}
	keyTop      = binding{keys: []string{"home", "g"}}
	keyBottom   = binding{keys: []string{"end", "G"}}
	keyOpen     = binding{keys: []string{"enter", "right", "l"}}
	keyBack     = binding{keys: []string{"esc", "backspace", "left", "h"}}
)

// A shortcut is one entry of the help the footer shows: the keys as
// they are spelled to the reader, and what they do, said in full and
// again in a word. A bar too narrow for the full wording shows the
// short one rather than dropping shortcuts off the end, because a
// shortcut nobody is told about might as well not be bound.
//
// The keys a reader needs first come first: whatever will not fit the
// bar in any wording is dropped from the end.
type shortcut struct {
	keys string
	help string
	// brief says the same in a word. Empty means help is already as
	// short as it goes.
	brief string
}

var shortcuts = []shortcut{
	{keys: "↑/k", help: "up"},
	{keys: "↓/j", help: "down"},
	{keys: "enter/→", help: "open", brief: "open"},
	{keys: "esc/←", help: "back"},
	{keys: "tab", help: "value", brief: "value"},
	{keys: "g/G", help: "top/bottom", brief: "ends"},
	{keys: "q", help: "quit"},
}

// A spelling is one way of writing a shortcut down: the keys as they
// are shown, and the words saying what they do. The words are what a
// narrow pane gives up, never the keys.
type spelling struct {
	keys string
	help string
}

// text is the spelling as the reader sees it, unstyled. The footer
// measures it in cells rather than in bytes, which is what keeps
// keys like "enter/→" from counting for twice their width.
func (s spelling) text() string {
	if s.help == "" {
		return s.keys
	}

	return s.keys + " " + s.help
}

// spellingCount is how many lengths a shortcut can be written at.
const spellingCount = 3

// shortcutGap separates two shortcuts along the footer.
const shortcutGap = "  "

// spell writes the shortcut out at one of those lengths, longest
// first: the full wording, the one-word one, and the bare keys.
func (s shortcut) spell(length int) spelling {
	switch length {
	case 0:
		return spelling{keys: s.keys, help: s.help}

	case 1:
		if s.brief == "" {
			return spelling{keys: s.keys, help: s.help}
		}

		return spelling{keys: s.keys, help: s.brief}

	default:
		return spelling{keys: s.keys}
	}
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

// footer renders the bar below the columns: every shortcut, at the
// fullest wording the width holds them all at. It is framed like a
// column, so the keys read as something the browser offers rather
// than as a line of text somebody left at the foot of the screen.
func (m Model) footer() string {
	room := max(m.width-boxBorder-boxPadding, 0)

	var line string
	for length := range spellingCount {
		var shown int
		line, shown = m.packShortcuts(length, room)
		if shown == len(shortcuts) {
			break
		}
	}

	return m.styles.footer.Width(max(m.width-boxBorder, 0)).Render(line)
}

// packShortcuts lays the shortcuts along a bar of the given room,
// written at the given length, and says how many of them it got in.
// What is left over is dropped from the end rather than wrapped into
// a second line the footer does not have.
func (m Model) packShortcuts(length, room int) (string, int) {
	var (
		line  string
		used  int
		shown int
	)
	for _, s := range shortcuts {
		spelled := s.spell(length)
		gap := shortcutGap
		if used == 0 {
			gap = ""
		}
		if used+lipgloss.Width(gap+spelled.text()) > room {
			break
		}
		line += gap + m.renderShortcut(spelled)
		used += lipgloss.Width(gap + spelled.text())
		shown++
	}

	return line, shown
}

// renderShortcut styles one shortcut: its keys stand out, the words
// saying what they do do not.
func (m Model) renderShortcut(s spelling) string {
	out := m.styles.shortcutKey.Render(s.keys)
	if s.help != "" {
		out += " " + m.styles.shortcut.Render(s.help)
	}

	return out
}
