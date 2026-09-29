package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Br0ce/boltcutter/tree"
)

var _ tea.Model = Model{}

// focus says what the keys move: the cursor of the current listing, or
// the viewport of a value too tall for the column showing it.
type focus int

const (
	focusListing focus = iota
	focusValue
)

// Layout constants. The two listing columns are sized as a share of the
// terminal width so the preview keeps the room it needs on wide screens.
const (
	// columnCount is how many columns are drawn, however deep the
	// browser is. They are filled from the left, so which of them
	// holds what changes but their number does not.
	columnCount = 3

	sideColumnRatio = 4 // the two listing columns take a quarter each.
	minColumnWidth  = 12
	minColumnHeight = 3

	// The header and the footer are framed like a column, so each
	// takes a line of content plus its border.
	headerHeight = 1 + boxBorder
	footerHeight = 1 + boxBorder
	// boxBorder and boxPadding are the width and height the border and
	// the padding add on top of the content of a framed box. The
	// columns, the header and the footer are each drawn in one.
	boxBorder  = 2
	boxPadding = 2
)

// Model is the bubbletea model of the browser. It keeps one column per
// level from the root down to the node that is open, and shows the last
// two of them beside a preview of what the cursor is on.
type Model struct {
	tree   tree.Tree
	styles styles
	// dbPath is the database file being browsed, shown in the header.
	dbPath string

	// columns is never empty: columns[0] lists the database root and
	// the last one holds the cursor.
	columns []*column

	preview preview

	focus  focus
	width  int
	height int
	err    error
}

// Init satisfies tea.Model. The first columns are already open, so
// there is nothing to do before the first frame.
func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.resize()

		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case keyMatches(msg, keyQuit):
		return m, tea.Quit

	case keyMatches(msg, keyFocus):
		m.toggleFocus()

	case keyMatches(msg, keyUp):
		m.scroll(-1)

	case keyMatches(msg, keyDown):
		m.scroll(1)

	case keyMatches(msg, keyPageUp):
		m.scroll(-m.listHeight())

	case keyMatches(msg, keyPageDown):
		m.scroll(m.listHeight())

	case keyMatches(msg, keyTop):
		m.scrollToStart()

	case keyMatches(msg, keyBottom):
		m.scrollToEnd()

	case keyMatches(msg, keyOpen):
		m.descend()

	case keyMatches(msg, keyBack):
		m.ascend()
	}

	return m, nil
}

// current is the column the cursor is in.
func (m Model) current() *column {
	return m.columns[len(m.columns)-1]
}

// path names the node that is open.
func (m Model) path() tree.Path {
	return m.current().path
}

// descend opens the node under the cursor. Its listing is already in
// the preview column, so descending shifts the columns along and reads
// nothing a second time. With a leaf under the cursor there is no
// listing to take, and the selection is left alone.
func (m *Model) descend() {
	if m.preview.column == nil {
		return
	}
	m.columns = append(m.columns, m.preview.column)
	// The column has moved into the browser and is no longer the
	// preview's to close.
	m.preview.column = nil
	m.focus = focusListing
	m.loadPreview()
}

// ascend leaves the current node for the one holding it. The parent's
// cursor never left the entry we descended through, so the cursor lands
// where it started.
func (m *Model) ascend() {
	if len(m.columns) == 1 {
		return
	}
	m.closeColumn(m.current())
	m.columns = m.columns[:len(m.columns)-1]
	m.focus = focusListing
	m.loadPreview()
}

// toggleFocus switches between moving the cursor and scrolling a value
// too tall for the preview column. There is nothing to scroll unless
// the preview is holding a value.
func (m *Model) toggleFocus() {
	if m.focus == focusListing && len(m.preview.lines) > 0 {
		m.focus = focusValue

		return
	}
	m.focus = focusListing
}

// scroll moves the cursor of the current column, or the preview
// viewport, by delta lines.
func (m *Model) scroll(delta int) {
	if m.focus == focusValue {
		m.preview.offset = clampOffset(m.preview.offset+delta, len(m.preview.lines), m.listHeight())

		return
	}
	m.current().move(delta)
	m.loadPreview()
}

func (m *Model) scrollToStart() {
	if m.focus == focusValue {
		m.preview.offset = 0

		return
	}
	m.current().top()
	m.loadPreview()
}

func (m *Model) scrollToEnd() {
	if m.focus == focusValue {
		m.preview.offset = clampOffset(len(m.preview.lines), len(m.preview.lines), m.listHeight())

		return
	}
	m.current().bottom()
	m.loadPreview()
}

// loadPreview fills the preview column from the entry under the
// cursor: the listing of a node, or the value of a leaf. It is the one
// place in the program that asks what an entry is.
func (m *Model) loadPreview() {
	m.closePreview()
	m.err = nil

	entry, ok := m.current().selected()
	if !ok {
		return
	}
	path := m.path().Child(entry.Name)

	switch entry.Kind {
	case tree.Node:
		column, err := newColumn(m.tree, path, m.listHeight())
		if err != nil {
			m.err = err

			return
		}
		m.preview.column = column

	case tree.Leaf:
		raw, size, err := m.tree.Read(path, previewLimit)
		if err != nil {
			m.err = err

			return
		}
		m.preview.setValue(raw, size)
	}

	// The focus belongs on the listing unless there is a value under
	// it to scroll.
	if len(m.preview.lines) == 0 {
		m.focus = focusListing
	}
}

// closePreview empties the preview column, releasing the listing it may
// have been holding.
func (m *Model) closePreview() {
	if err := m.preview.close(); err != nil {
		m.err = err
	}
	m.preview = preview{}
}

// closeColumn releases a column, reporting a failure rather than
// swallowing it: a listing that will not close is a leak we would
// rather hear about.
func (m *Model) closeColumn(c *column) {
	if err := c.Close(); err != nil {
		m.err = err
	}
}

// Close releases every listing the browser holds. The Model is of no
// further use afterwards.
func (m *Model) Close() {
	m.closePreview()
	for _, column := range m.columns {
		m.closeColumn(column)
	}
	m.columns = nil
}

// resize propagates the terminal size to the columns.
func (m *Model) resize() {
	height := m.listHeight()
	for _, column := range m.columns {
		column.setHeight(height)
	}
	m.preview.column.setHeight(height)
	m.preview.offset = clampOffset(m.preview.offset, len(m.preview.lines), height)
}

// columnHeight is the outer height of a column, i.e. what is left
// between the header and the footer.
func (m Model) columnHeight() int {
	return max(m.height-headerHeight-footerHeight, minColumnHeight)
}

// listHeight is how many item lines fit inside a column.
func (m Model) listHeight() int {
	return max(m.columnHeight()-boxBorder, 1)
}

// hasParent reports whether we have dived into a node, so there is a
// parent listing to show alongside the current one.
func (m Model) hasParent() bool {
	return len(m.columns) > 1
}

// columnWidths returns the outer widths of the three columns, left to
// right. They never change as the columns shift, so nothing jumps
// around while browsing.
func (m Model) columnWidths() [columnCount]int {
	side := max(m.width/sideColumnRatio, minColumnWidth)
	preview := max(m.width-2*side, minColumnWidth)

	return [columnCount]int{side, side, preview}
}

// clampOffset keeps a viewport offset within a document of n lines shown
// through a window of the given height.
func clampOffset(offset, n, height int) int {
	return min(max(offset, 0), max(n-height, 0))
}
