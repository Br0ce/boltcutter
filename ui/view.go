package ui

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"

	"github.com/Br0ce/boltcutter/tree"
	"github.com/Br0ce/boltcutter/value"
)

const (
	// ellipsis marks text that did not fit the width it was given.
	ellipsis = "…"
	// unprintable stands in for a character a terminal cannot be
	// shown, and replacement for a byte that is no character at all.
	unprintable = "·"
	replacement = "\uFFFD"
)

// cursorStyle says how a column marks the entry under its cursor.
type cursorStyle int

const (
	// cursorHidden is the preview column: a listing nobody is in yet, so
	// marking a cursor in it would promise a place we are not.
	cursorHidden cursorStyle = iota
	// cursorIdle is a parent column: the cursor is shown, dimmed,
	// because it says which entry we came down through.
	cursorIdle
	// cursorActive is the column the keys are moving.
	cursorActive
)

// View renders the header, the three columns side by side, and the
// shortcut bar.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		// The first frame arrives before the terminal size does.
		return ""
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		m.header(),
		lipgloss.JoinHorizontal(lipgloss.Top, m.renderColumns()...),
		m.footer(),
	)
}

// renderColumns draws the three columns, left to right: the listing we
// came down through, the listing the cursor is in, and the preview of
// what the cursor is on.
//
// They are filled from the left, so at the database root, where there
// is no parent, everything shifts one column along and the empty one
// is the rightmost. A gap between a listing and its own preview would
// read as two unrelated things rather than one.
func (m Model) renderColumns() []string {
	widths := m.columnWidths()
	columns := make([]string, 0, columnCount)
	// Each one is drawn at the width of the next column still free,
	// which is all that filling from the left amounts to.
	width := func() int { return widths[len(columns)] }

	if m.hasParent() {
		parent := m.columns[len(m.columns)-2]
		columns = append(columns, m.renderColumn(cursorIdle, parent, width(), ""))
	}
	columns = append(columns, m.renderColumn(m.listingCursor(), m.current(), width(), m.emptyListing()))
	columns = append(columns, m.renderPreview(width()))
	for len(columns) < columnCount {
		columns = append(columns, m.renderBox(false, width(), nil))
	}

	return columns
}

// listingCursor says how the current column marks its cursor: the mark
// belongs to whatever the keys are moving, which is the listing unless
// the focus has been handed to a value.
func (m Model) listingCursor() cursorStyle {
	if m.focus == focusListing {
		return cursorActive
	}

	return cursorIdle
}

// emptyListing is the placeholder for a node that holds nothing. An
// empty database is worth telling apart from an empty bucket: one of
// them means the file has nothing in it at all.
func (m Model) emptyListing() string {
	if m.hasParent() {
		return "empty bucket"
	}

	return "empty database"
}

// renderPreview draws what the cursor is on: the listing of a node, the
// value of a leaf, or a note saying why neither is there to show.
func (m Model) renderPreview(width int) string {
	content := max(width-boxBorder-boxPadding, 1)
	focused := m.focus == focusValue

	if m.err != nil {
		return m.renderBox(focused, width, []string{
			m.styles.errorText.Render(truncate(m.err.Error(), content)),
		})
	}
	if m.preview.column != nil {
		return m.renderColumn(cursorHidden, m.preview.column, width, "empty bucket")
	}
	if m.preview.note != "" {
		return m.renderBox(focused, width, []string{
			m.styles.empty.Render(truncate(m.preview.note, content)),
		})
	}

	end := min(m.preview.offset+m.listHeight(), len(m.preview.lines))
	lines := make([]string, 0, max(end-m.preview.offset, 0))
	for _, line := range m.preview.lines[m.preview.offset:end] {
		lines = append(lines, m.styles.renderLine(line, content))
	}

	return m.renderBox(focused, width, lines)
}

// renderLine draws one line of a value, styling every token by its
// kind and marking a line too wide for the column with an ellipsis.
// Styling and cutting go together: the styles wrap the text in escape
// sequences that a later cut must not run into.
func (s styles) renderLine(line value.Line, width int) string {
	if width <= 0 {
		return ""
	}
	// Room for the ellipsis has to be kept before the first token is
	// rendered, so a line that does not fit gives up a cell up front.
	// Otherwise a cut falling between two tokens would leave the line
	// looking complete.
	budget := width
	cut := lipgloss.Width(line.Text()) > width
	if cut {
		budget--
	}

	var out strings.Builder
	for _, token := range line {
		if budget <= 0 {
			break
		}
		text := trim(token.Text, budget)
		out.WriteString(s.token(token.Kind).Render(text))
		budget -= lipgloss.Width(text)
	}
	if cut {
		out.WriteString(ellipsis)
	}

	return out.String()
}

// renderColumn draws a column with its cursor. empty is the
// placeholder shown when the node holds nothing.
func (m Model) renderColumn(style cursorStyle, c *column, width int, empty string) string {
	content := max(width-boxBorder-boxPadding, 1)
	focused := style == cursorActive
	if len(c.items) == 0 {
		return m.renderBox(focused, width, []string{m.styles.empty.Render(truncate(empty, content))})
	}

	lines := make([]string, 0, len(c.items))
	for i, item := range c.items {
		lines = append(lines, m.renderItem(style, item, i == c.cursor, content))
	}

	return m.renderBox(focused, width, lines)
}

// renderItem draws a single listing row. Nodes carry a trailing slash,
// the way a directory does, so they are told apart from leaves at a
// glance.
func (m Model) renderItem(style cursorStyle, entry tree.Entry, atCursor bool, width int) string {
	name := printable(entry.Name)
	prefix := "  "
	itemStyle := m.styles.item
	if entry.Kind == tree.Node {
		name += "/"
		itemStyle = m.styles.bucketItem
	}
	if atCursor && style != cursorHidden {
		prefix = "\u203a "
		itemStyle = m.styles.activeItem
		if style == cursorActive {
			itemStyle = m.styles.cursorItem
		}
	}

	return itemStyle.Width(width).Render(prefix + truncate(name, width-len(prefix)))
}

// renderBox frames content lines in a bordered box of the given outer
// width. The columns are drawn in one, and so is a column left empty.
func (m Model) renderBox(focused bool, width int, content []string) string {
	style := m.styles.box
	if focused {
		style = m.styles.activeBox
	}

	return style.
		Width(max(width-boxBorder, 1)).
		Height(max(m.columnHeight()-boxBorder, 1)).
		Render(strings.Join(content, "\n"))
}

// truncate shortens s to at most width cells, marking the cut with an
// ellipsis.
func truncate(s string, width int) string {
	if width <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= width {
		return s
	}

	// The ellipsis takes a cell of its own.
	return trim(s, width-1) + ellipsis
}

// trim shortens s to at most width cells, cutting it without a mark.
func trim(s string, width int) string {
	if width <= 0 {
		return ""
	}
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes)) > width {
		runes = runes[:len(runes)-1]
	}

	return string(runes)
}

// printable makes a raw name safe to show. A key of a bbolt database is
// any byte string at all, so it may hold control characters, escape
// sequences, or bytes that are no text in the first place: shown as
// they are, one key could rewrite the screen around it.
func printable(name string) string {
	var out strings.Builder
	for _, r := range name {
		switch {
		case r == utf8.RuneError:
			// A byte that decodes to nothing, or a replacement
			// character that was stored as one. Either way it is not
			// a character we can show.
			out.WriteString(replacement)

		case unicode.IsPrint(r):
			out.WriteRune(r)

		default:
			out.WriteString(unprintable)
		}
	}

	return out.String()
}
