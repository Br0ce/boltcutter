package ui

import "github.com/Br0ce/boltcutter/store"

// list is a vertically scrolling list of bucket entries with a cursor. It keeps
// the cursor inside the visible window by adjusting the scroll offset
// whenever the cursor or the window height moves.
type list struct {
	items  []store.Entry
	cursor int
	offset int
	height int
}

// setItems replaces the content of the list and clamps the cursor, so a
// reload that shortens the list never leaves the cursor dangling.
func (l *list) setItems(items []store.Entry) {
	l.items = items
	l.setCursor(l.cursor)
}

// setHeight sets how many items are visible at once.
func (l *list) setHeight(height int) {
	l.height = max(height, 0)
	l.scrollToCursor()
}

// selected returns the entry under the cursor, and false if the list is
// empty.
func (l *list) selected() (store.Entry, bool) {
	if len(l.items) == 0 {
		return store.Entry{}, false
	}

	return l.items[l.cursor], true
}

// move shifts the cursor by delta, stopping at either end of the list.
func (l *list) move(delta int) {
	l.setCursor(l.cursor + delta)
}

// setCursor moves the cursor to index, clamped to the list bounds.
func (l *list) setCursor(index int) {
	if len(l.items) == 0 {
		l.cursor, l.offset = 0, 0

		return
	}
	l.cursor = min(max(index, 0), len(l.items)-1)
	l.scrollToCursor()
}

// scrollToCursor moves the visible window the shortest distance that
// brings the cursor back into view.
func (l *list) scrollToCursor() {
	if l.height <= 0 {
		l.offset = 0

		return
	}
	l.offset = min(l.offset, l.cursor)
	l.offset = max(l.offset, l.cursor-l.height+1)
	// Never scroll past the end while items still fit above.
	l.offset = min(l.offset, max(len(l.items)-l.height, 0))
	l.offset = max(l.offset, 0)
}

// window returns the currently visible items and the index of the first
// of them within the list.
func (l *list) window() ([]store.Entry, int) {
	if l.height <= 0 || len(l.items) == 0 {
		return nil, 0
	}
	end := min(l.offset+l.height, len(l.items))

	return l.items[l.offset:end], l.offset
}
