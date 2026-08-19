package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// View renders the header, the three panes side by side, and the
// shortcut bar.
func (m Model) View() string {
	if m.width == 0 || m.height == 0 {
		// The first frame arrives before the terminal size does.
		return ""
	}
	widths := m.paneWidths()
	left, middle := m.renderListings(widths)
	panes := lipgloss.JoinHorizontal(
		lipgloss.Top,
		left,
		middle,
		m.renderPreview(widths[panePreview]),
	)

	return lipgloss.JoinVertical(lipgloss.Left, m.header(), panes, m.footer())
}

// renderListings draws the two listing columns. The listings fill the
// screen from the left, so at the database root the cursor is in the left
// column and the middle waits for the bucket to dive into. Once we have
// dived, the parent takes the left column and the cursor moves to the
// middle, which is where it stays however deep we go.
func (m Model) renderListings(widths [paneCount]int) (left, middle string) {
	// The listing columns share one cursor, so they are focused
	// together, as opposed to the value in the preview.
	focused := m.focus == paneCurrent

	if !m.hasParent() {
		return m.renderList(focused, m.current().list, widths[paneParent], "empty database"),
			m.renderPane(false, widths[paneCurrent], nil)
	}
	parent := m.levels[len(m.levels)-2]

	return m.renderList(false, parent.list, widths[paneParent], ""),
		m.renderList(focused, m.current().list, widths[paneCurrent], "empty bucket")
}

// renderPreview draws the JSON value of the selected key. The pane is
// reserved for values: with a bucket under the cursor, or a value that is
// not JSON, it stays empty.
func (m Model) renderPreview(width int) string {
	content := max(width-paneBorder-panePadding, 1)
	focused := m.focus == panePreview

	if m.err != nil {
		return m.renderPane(focused, width, []string{
			m.styles.errorText.Render(truncate(m.err.Error(), content)),
		})
	}

	end := min(m.valueOffset+m.listHeight(), len(m.valueLines))
	lines := make([]string, 0, end-m.valueOffset)
	for _, line := range m.valueLines[m.valueOffset:end] {
		// Highlight after truncating: the styles wrap the text in
		// escape sequences that truncation must not cut into.
		lines = append(lines, m.styles.highlightJSON(truncate(line, content)))
	}

	return m.renderPane(focused, width, lines)
}

// renderList draws a listing with its cursor. empty is the placeholder
// shown when the bucket holds nothing.
func (m Model) renderList(focused bool, l list, width int, empty string) string {
	content := max(width-paneBorder-panePadding, 1)
	items, first := l.window()
	if len(items) == 0 {
		return m.renderPane(focused, width, []string{m.styles.empty.Render(truncate(empty, content))})
	}

	lines := make([]string, 0, len(items))
	for i, item := range items {
		lines = append(lines, m.renderItem(focused, item, first+i == l.cursor, content))
	}

	return m.renderPane(focused, width, lines)
}

// renderItem draws a single listing row. Buckets carry a trailing slash,
// the way a directory does, so they are told apart from keys at a glance.
func (m Model) renderItem(focused bool, entry Entry, atCursor bool, width int) string {
	name := entry.Name
	if entry.Bucket {
		name += "/"
	}

	prefix := "  "
	style := m.styles.item
	if entry.Bucket {
		style = m.styles.bucketItem
	}
	if atCursor {
		prefix = "› "
		style = m.styles.activeItem
		if focused {
			style = m.styles.cursorItem
		}
	}

	return style.Width(width).Render(prefix + truncate(name, width-len(prefix)))
}

// renderPane frames content lines in a bordered box of the given outer
// width.
func (m Model) renderPane(focused bool, width int, content []string) string {
	style := m.styles.pane
	if focused {
		style = m.styles.activePane
	}

	return style.
		Width(max(width-paneBorder, 1)).
		Height(max(m.paneHeight()-paneBorder, 1)).
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
	runes := []rune(s)
	for len(runes) > 0 && lipgloss.Width(string(runes))+1 > width {
		runes = runes[:len(runes)-1]
	}

	return string(runes) + "…"
}
