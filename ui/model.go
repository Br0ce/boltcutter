package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Br0ce/boltcutter/store"
)

// pane identifies one of the three columns.
type pane int

const (
	// paneParent shows the bucket the current listing sits in. It is
	// empty at the database root.
	paneParent pane = iota
	// paneCurrent shows the bucket that is open, and holds the cursor.
	paneCurrent
	// panePreview is reserved for the JSON value of the selected key.
	panePreview

	paneCount = 3
)

// Layout constants. The two left columns are sized as a share of the
// terminal width so the preview keeps the room it needs on wide screens.
const (
	sidePaneRatio = 4 // the parent and current columns take a quarter each.
	minPaneWidth  = 12
	minPaneHeight = 3
	// The header and the footer are boxed like the panes, so each takes
	// a line of content plus its border.
	headerHeight = 1 + paneBorder
	footerHeight = 1 + paneBorder
	// paneBorder and panePadding are the width and height the border
	// and the padding of a pane add on top of its content.
	paneBorder  = 2
	panePadding = 2
)

// level is one bucket listing on the way down from the root.
type level struct {
	// path names the bucket this listing belongs to; empty for the
	// database root.
	path []string
	list list
}

// Model is the bubbletea model of the browser. It keeps every listing
// from the root down to the bucket that is open, and shows the last two
// of them next to the value of the selected key.
type Model struct {
	store  Store
	styles styles
	// dbPath is the database file being browsed, shown in the header.
	dbPath string

	// levels is never empty: levels[0] is the database root and the
	// last element is the listing the cursor is in.
	levels []level

	// valueLines is the rendered value of the selected key, empty
	// whenever the cursor is on a bucket.
	valueLines  []string
	valueOffset int

	focus  pane
	width  int
	height int
	err    error
}

// Init satisfies tea.Model. The first listing is already loaded by New,
// so there is nothing to do before the first frame.
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

//nolint:gocyclo // A flat switch over key bindings reads better than a dispatch table.
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

	case keyMatches(msg, keyReload):
		m.reload()
	}

	return m, nil
}

// current is the listing the cursor is in.
func (m *Model) current() *level {
	return &m.levels[len(m.levels)-1]
}

// path is the bucket that is currently open.
func (m Model) path() []string {
	return m.levels[len(m.levels)-1].path
}

// toggleFocus switches between moving the cursor and scrolling a value
// too tall for the preview pane.
func (m *Model) toggleFocus() {
	if m.focus == paneCurrent {
		m.focus = panePreview

		return
	}
	m.focus = paneCurrent
}

// scroll moves the cursor of the current listing, or the preview
// viewport, by delta lines.
func (m *Model) scroll(delta int) {
	if m.focus == panePreview {
		m.valueOffset = clampOffset(m.valueOffset+delta, len(m.valueLines), m.listHeight())

		return
	}
	m.current().list.move(delta)
	m.loadValue()
}

func (m *Model) scrollToStart() {
	if m.focus == panePreview {
		m.valueOffset = 0

		return
	}
	m.current().list.setCursor(0)
	m.loadValue()
}

func (m *Model) scrollToEnd() {
	if m.focus == panePreview {
		m.valueOffset = clampOffset(len(m.valueLines), len(m.valueLines), m.listHeight())

		return
	}
	m.current().list.setCursor(len(m.current().list.items) - 1)
	m.loadValue()
}

// descend opens the selected bucket, shifting the columns one to the
// left. Keys cannot be descended into, so the selection is left alone.
func (m *Model) descend() {
	entry, ok := m.current().list.selected()
	if !ok || !entry.Bucket {
		return
	}
	path := append(append([]string{}, m.path()...), entry.Name)
	entries, err := m.store.Entries(path)
	if err != nil {
		m.err = err

		return
	}
	m.err = nil
	next := level{path: path}
	next.list.setHeight(m.listHeight())
	next.list.setItems(entries)
	m.levels = append(m.levels, next)
	m.focus = paneCurrent
	m.loadValue()
}

// ascend leaves the current bucket for its parent, shifting the columns
// one to the right. The parent's cursor still sits on the bucket that was
// left, so the cursor lands where it started.
func (m *Model) ascend() {
	if len(m.levels) == 1 {
		return
	}
	m.levels = m.levels[:len(m.levels)-1]
	m.focus = paneCurrent
	m.loadValue()
}

// reload re-reads every open listing from the store, keeping the cursors
// on the entries they were on. Buckets that disappeared underneath the UI
// drop the levels below them. Errors are shown rather than fatal, so a
// database that changes stays browsable.
func (m *Model) reload() {
	// Remember where the cursors were, by name, before re-reading.
	path := m.path()
	selected := ""
	if entry, ok := m.current().list.selected(); ok {
		selected = entry.Name
	}

	levels := make([]level, 0, len(m.levels))
	for i := 0; i <= len(path); i++ {
		if i > 0 && !containsBucket(levels[i-1].list.items, path[i-1]) {
			// The bucket vanished underneath us; stop at its parent.
			break
		}
		entries, err := m.store.Entries(path[:i])
		if err != nil {
			m.err = err

			return
		}
		next := level{path: path[:i]}
		next.list.setHeight(m.listHeight())
		next.list.setItems(entries)
		if i < len(path) {
			next.list.setCursor(indexOf(entries, path[i]))
		}
		levels = append(levels, next)
	}
	m.err = nil
	m.levels = levels
	m.current().list.setCursor(indexOf(m.current().list.items, selected))
	m.loadValue()
}

// loadValue fills the preview pane with the value of the selected key.
// The pane is reserved for values, so a bucket under the cursor leaves it
// empty.
func (m *Model) loadValue() {
	m.valueLines, m.valueOffset = nil, 0

	entry, ok := m.current().list.selected()
	if !ok || entry.Bucket {
		return
	}

	value, err := m.store.Value(m.path(), entry.Name)
	if err != nil {
		m.err = err

		return
	}
	m.err, m.valueLines = nil, formatValue(value)
}

// resize propagates the terminal size to the listings.
func (m *Model) resize() {
	height := m.listHeight()
	for i := range m.levels {
		m.levels[i].list.setHeight(height)
	}
	m.valueOffset = clampOffset(m.valueOffset, len(m.valueLines), height)
}

// paneHeight is the outer height of a pane, i.e. what is left between the
// header and the footer.
func (m Model) paneHeight() int {
	return max(m.height-headerHeight-footerHeight, minPaneHeight)
}

// listHeight is how many item lines fit inside a pane.
func (m Model) listHeight() int {
	return max(m.paneHeight()-paneBorder, 1)
}

// hasParent reports whether we have dived into a bucket, so the parent
// column has a listing to show.
func (m Model) hasParent() bool {
	return len(m.levels) > 1
}

// paneWidths returns the outer widths of the three panes, left to right.
// They never change as the columns shift, so nothing jumps around while
// browsing.
func (m Model) paneWidths() [paneCount]int {
	side := max(m.width/sidePaneRatio, minPaneWidth)
	preview := max(m.width-2*side, minPaneWidth)

	return [paneCount]int{side, side, preview}
}

// clampOffset keeps a viewport offset within a document of n lines shown
// through a window of the given height.
func clampOffset(offset, n, height int) int {
	return min(max(offset, 0), max(n-height, 0))
}

// indexOf returns the position of the entry named name, or 0 if it is
// absent.
func indexOf(entries []store.Entry, name string) int {
	for i, entry := range entries {
		if entry.Name == name {
			return i
		}
	}

	return 0
}

// containsBucket reports whether entries hold a nested bucket named name.
func containsBucket(entries []store.Entry, name string) bool {
	for _, entry := range entries {
		if entry.Name == name && entry.Bucket {
			return true
		}
	}

	return false
}

var _ tea.Model = Model{}
