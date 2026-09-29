package ui

import (
	"slices"

	"github.com/Br0ce/boltcutter/tree"
)

// column is one listing column of the browser: a window of at most
// height entries out of
// a tree.Listing, and a cursor inside that window.
//
// The listing is a cursor into the source rather than a copy of it, so
// a column holds only what it shows. Scrolling pulls entries in at one
// end of the window and drops them at the other, which is what lets a
// bucket of a million keys cost the same as one of ten.
//
// Every move places the listing itself before walking it, by seeking to
// an entry the window already holds. That costs a lookup and buys the
// methods here their independence: none of them has to know where
// another left the listing.
type column struct {
	// path names the node this lists, and is what a child's path is
	// built from.
	path    tree.Path
	listing tree.Listing

	// items is the window: the entries on screen, top to bottom, and
	// cursor is the one under the cursor among them.
	items  []tree.Entry
	cursor int
	height int
}

// newColumn lists the node at path in a window height entries tall,
// opened at the top of the listing.
func newColumn(t tree.Tree, path tree.Path, height int) (*column, error) {
	listing, err := t.Open(path)
	if err != nil {
		return nil, err
	}
	c := &column{path: path, listing: listing, height: max(height, 1)}
	c.top()

	return c, nil
}

// Close releases the listing. A nil column closes cleanly, so a slot
// that is not showing a listing needs no guard of its own.
func (c *column) Close() error {
	if c == nil || c.listing == nil {
		return nil
	}
	listing := c.listing
	c.listing = nil

	return listing.Close()
}

// selected returns the entry under the cursor, and false when the node
// holds nothing.
func (c *column) selected() (tree.Entry, bool) {
	if len(c.items) == 0 {
		return tree.Entry{}, false
	}

	return c.items[c.cursor], true
}

// selectedPath names the entry under the cursor.
func (c *column) selectedPath() (tree.Path, bool) {
	entry, ok := c.selected()
	if !ok {
		return nil, false
	}

	return c.path.Child(entry.Name), true
}

// top puts the window at the start of the listing, cursor on the first
// entry.
func (c *column) top() {
	c.place(c.listing.First())
}

// bottom puts the window at the end of the listing, cursor on the last
// entry. fill has nothing to pull in below it, so it fills above
// instead and carries the cursor down to the foot of the window.
func (c *column) bottom() {
	c.place(c.listing.Last())
}

// setHeight sets how many entries the window holds, keeping the entry
// under the cursor in it. A nil column resizes cleanly, so a slot that
// is not showing a listing needs no guard of its own.
func (c *column) setHeight(height int) {
	if c == nil {
		return
	}
	c.height = max(height, 1)

	entry, ok := c.selected()
	if !ok {
		c.top()

		return
	}
	c.focus(entry.Name)
}

// focus rebuilds the window around the entry named name and puts the
// cursor on it. A name that is not in the listing leaves the column at
// the top and reports false.
func (c *column) focus(name string) bool {
	entry, ok := c.listing.Seek(name)
	if !ok || entry.Name != name {
		c.top()

		return false
	}
	c.place(entry, true)

	return true
}

// move shifts the cursor by delta, scrolling the window when the cursor
// runs past either end of it, and stopping at the ends of the listing.
func (c *column) move(delta int) {
	if len(c.items) == 0 {
		return
	}
	switch target := c.cursor + delta; {
	case target < 0:
		c.scrollUp(-target)
		c.cursor = 0

	case target >= len(c.items):
		c.scrollDown(target - len(c.items) + 1)
		c.cursor = len(c.items) - 1

	default:
		c.cursor = target
	}
}

// place restarts the window on a single entry and fills it out. It
// takes what a move of the listing returned, so an empty node empties
// the column instead of leaving the last window behind.
func (c *column) place(entry tree.Entry, ok bool) {
	if !ok {
		c.items, c.cursor = nil, 0

		return
	}
	c.items, c.cursor = []tree.Entry{entry}, 0
	c.fill()
}

// fill pulls entries in until the window is full: below the cursor
// first, and above it for whatever the end of the listing left over, so
// a window at the foot of a listing is as full as one in the middle.
func (c *column) fill() {
	c.scrollDown(c.height - len(c.items))
	if room := c.height - len(c.items); room > 0 {
		c.scrollUp(room)
	}
}

// scrollDown pulls up to n entries in at the foot of the window,
// dropping as many from the head, and stops at the end of the listing.
func (c *column) scrollDown(n int) {
	if n <= 0 || len(c.items) == 0 {
		return
	}
	// Where an earlier move left the listing is none of this method's
	// business: the window says where to carry on from.
	c.listing.Seek(c.items[len(c.items)-1].Name)
	for range n {
		entry, ok := c.listing.Next()
		if !ok {
			break
		}
		c.items = append(c.items, entry)
	}
	if over := len(c.items) - c.height; over > 0 {
		c.items = c.items[over:]
		c.cursor = max(c.cursor-over, 0)
	}
}

// scrollUp pulls up to n entries in at the head of the window, dropping
// as many from the foot, and stops at the start of the listing.
func (c *column) scrollUp(n int) {
	if n <= 0 || len(c.items) == 0 {
		return
	}
	c.listing.Seek(c.items[0].Name)

	var ahead []tree.Entry
	for range n {
		entry, ok := c.listing.Prev()
		if !ok {
			break
		}
		ahead = append(ahead, entry)
	}
	if len(ahead) == 0 {
		return
	}
	// Prev walks backwards, so the entries arrived bottom-first.
	slices.Reverse(ahead)
	c.items = append(ahead, c.items...)
	c.cursor += len(ahead)

	if len(c.items) > c.height {
		c.items = c.items[:c.height]
		c.cursor = min(c.cursor, c.height-1)
	}
}
