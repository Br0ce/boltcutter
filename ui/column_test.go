package ui

import (
	"slices"
	"testing"

	"github.com/Br0ce/boltcutter/tree"
)

func TestColumnOpensAtTheTop(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)

	if got := names(c.items); !slices.Equal(got, []string{"k00", "k01", "k02", "k03", "k04"}) {
		t.Errorf("window = %v, want the first five entries", got)
	}
	if c.cursor != 0 {
		t.Errorf("cursor = %d, want 0", c.cursor)
	}
}

func TestColumnShorterThanItsWindow(t *testing.T) {
	t.Parallel()

	// Two entries in a window of five: the window holds what there is
	// and no more.
	c := openColumn(t, newFakeTree(), tree.Path{"users"}, 5)

	if got := names(c.items); !slices.Equal(got, []string{"user:001", "user:002"}) {
		t.Errorf("window = %v, want both entries", got)
	}
	c.move(10)
	if entry, _ := c.selected(); entry.Name != "user:002" {
		t.Errorf("cursor after moving past the end = %q, want \"user:002\"", entry.Name)
	}
	c.move(-10)
	if entry, _ := c.selected(); entry.Name != "user:001" {
		t.Errorf("cursor after moving past the start = %q, want \"user:001\"", entry.Name)
	}
}

func TestColumnOfNothing(t *testing.T) {
	t.Parallel()

	ft := newFakeTree()
	ft.nodes["empty"] = nil
	c := openColumn(t, ft, tree.Path{"empty"}, 5)

	if len(c.items) != 0 {
		t.Errorf("window = %v, want it empty", names(c.items))
	}
	if _, ok := c.selected(); ok {
		t.Error("an empty column has an entry selected, want none")
	}
	if _, ok := c.selectedPath(); ok {
		t.Error("an empty column has a selected path, want none")
	}
	// Moving about an empty column is allowed and does nothing.
	c.move(1)
	c.move(-1)
	c.bottom()
	if len(c.items) != 0 {
		t.Errorf("window after moving = %v, want it empty", names(c.items))
	}
}

func TestColumnScrollsTheWindow(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)

	// Moving inside the window does not scroll it.
	c.move(4)
	if got := names(c.items); !slices.Equal(got, []string{"k00", "k01", "k02", "k03", "k04"}) {
		t.Fatalf("window after moving inside it = %v, want it unmoved", got)
	}
	if c.cursor != 4 {
		t.Fatalf("cursor = %d, want 4", c.cursor)
	}

	// Moving past the foot pulls one entry in and drops one off the
	// head, leaving the cursor at the foot.
	c.move(1)
	if got := names(c.items); !slices.Equal(got, []string{"k01", "k02", "k03", "k04", "k05"}) {
		t.Errorf("window after scrolling down = %v, want it moved by one", got)
	}
	if c.cursor != 4 {
		t.Errorf("cursor after scrolling down = %d, want it at the foot", c.cursor)
	}

	// And back the other way.
	c.move(-4)
	if c.cursor != 0 {
		t.Fatalf("cursor = %d, want 0", c.cursor)
	}
	c.move(-1)
	if got := names(c.items); !slices.Equal(got, []string{"k00", "k01", "k02", "k03", "k04"}) {
		t.Errorf("window after scrolling up = %v, want it back where it started", got)
	}
	if c.cursor != 0 {
		t.Errorf("cursor after scrolling up = %d, want it at the head", c.cursor)
	}
}

func TestColumnPagesByMoreThanAWindow(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)

	c.move(12)
	if entry, _ := c.selected(); entry.Name != "k12" {
		t.Errorf("cursor after a jump of twelve = %q, want \"k12\"", entry.Name)
	}
	if got := names(c.items); !slices.Equal(got, []string{"k08", "k09", "k10", "k11", "k12"}) {
		t.Errorf("window = %v, want the five entries ending at k12", got)
	}

	c.move(-7)
	if entry, _ := c.selected(); entry.Name != "k05" {
		t.Errorf("cursor after a jump back of seven = %q, want \"k05\"", entry.Name)
	}
}

func TestColumnBottomFillsTheWindowAbove(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)
	c.bottom()

	// The listing ends, so the window fills upwards and the cursor
	// sits at its foot rather than leaving four empty rows.
	if got := names(c.items); !slices.Equal(got, []string{"k15", "k16", "k17", "k18", "k19"}) {
		t.Errorf("window at the bottom = %v, want the last five entries", got)
	}
	if c.cursor != 4 {
		t.Errorf("cursor at the bottom = %d, want 4", c.cursor)
	}

	c.top()
	if got := names(c.items); !slices.Equal(got, []string{"k00", "k01", "k02", "k03", "k04"}) {
		t.Errorf("window at the top = %v, want the first five entries", got)
	}
	if c.cursor != 0 {
		t.Errorf("cursor at the top = %d, want 0", c.cursor)
	}
}

func TestColumnKeepsTheCursorWhenResized(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)
	c.move(12)

	c.setHeight(3)
	if entry, _ := c.selected(); entry.Name != "k12" {
		t.Errorf("cursor after shrinking = %q, want \"k12\"", entry.Name)
	}
	if len(c.items) != 3 {
		t.Errorf("window after shrinking = %v, want three entries", names(c.items))
	}

	c.setHeight(8)
	if entry, _ := c.selected(); entry.Name != "k12" {
		t.Errorf("cursor after growing = %q, want \"k12\"", entry.Name)
	}
	if len(c.items) != 8 {
		t.Errorf("window after growing = %v, want eight entries", names(c.items))
	}
}

func TestColumnResizedAtTheEndStaysFull(t *testing.T) {
	t.Parallel()

	// Growing the window with the cursor on the last entry has to
	// fill upwards, or the column would show a short listing.
	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)
	c.bottom()
	c.setHeight(8)

	if got := names(c.items); len(got) != 8 || got[7] != "k19" {
		t.Errorf("window = %v, want eight entries ending at k19", got)
	}
	if entry, _ := c.selected(); entry.Name != "k19" {
		t.Errorf("cursor = %q, want \"k19\"", entry.Name)
	}
}

func TestColumnFocusesAName(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree().withSequence("seq", 20), tree.Path{"seq"}, 5)

	if !c.focus("k07") {
		t.Fatal("focusing an entry that is there reported false")
	}
	if entry, _ := c.selected(); entry.Name != "k07" {
		t.Errorf("cursor = %q, want \"k07\"", entry.Name)
	}

	// A name that is not there is not silently rounded to its
	// neighbour: the column says so and goes back to the top.
	if c.focus("nope") {
		t.Error("focusing an entry that is not there reported true")
	}
	if entry, _ := c.selected(); entry.Name != "k00" {
		t.Errorf("cursor after a failed focus = %q, want \"k00\"", entry.Name)
	}
}

func TestColumnSelectedPath(t *testing.T) {
	t.Parallel()

	c := openColumn(t, newFakeTree(), tree.Path{"config"}, 5)

	path, ok := c.selectedPath()
	if !ok {
		t.Fatal("no entry selected, want the first")
	}
	if want := (tree.Path{"config", "flags"}); !slices.Equal(path, want) {
		t.Errorf("selected path = %q, want %q", path, want)
	}
}

func TestColumnCloseIsNilSafe(t *testing.T) {
	t.Parallel()

	var c *column
	if err := c.Close(); err != nil {
		t.Errorf("closing a nil column: %v", err)
	}
	// A slot holding no listing is resized like any other.
	c.setHeight(10)
}
