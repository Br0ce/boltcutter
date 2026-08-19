package ui

import (
	"slices"
	"testing"
)

// items returns n entries named "a", "b", …
func items(n int) []Entry {
	out := make([]Entry, n)
	for i := range out {
		out[i] = Entry{Name: string(rune('a' + i))}
	}

	return out
}

// names returns the entry names of a window, for comparison in tests.
func names(entries []Entry) []string {
	if entries == nil {
		return nil
	}
	out := make([]string, len(entries))
	for i, entry := range entries {
		out[i] = entry.Name
	}

	return out
}

func TestListMoveClampsToBounds(t *testing.T) {
	t.Parallel()

	l := list{height: 3}
	l.setItems(items(4))

	l.move(-1)
	if l.cursor != 0 {
		t.Errorf("cursor after moving up from the top = %d, want 0", l.cursor)
	}

	l.move(100)
	if l.cursor != 3 {
		t.Errorf("cursor after moving past the end = %d, want 3", l.cursor)
	}
}

func TestListEmpty(t *testing.T) {
	t.Parallel()

	l := list{height: 3}
	l.setItems(nil)

	if _, ok := l.selected(); ok {
		t.Error("selected() reported an item on an empty list")
	}
	l.move(2)
	if l.cursor != 0 || l.offset != 0 {
		t.Errorf("cursor, offset = %d, %d, want 0, 0", l.cursor, l.offset)
	}
}

func TestListSetItemsClampsCursor(t *testing.T) {
	t.Parallel()

	l := list{height: 3}
	l.setItems(items(5))
	l.setCursor(4)

	l.setItems(items(2))

	if l.cursor != 1 {
		t.Errorf("cursor after shrinking the list = %d, want 1", l.cursor)
	}
	got, ok := l.selected()
	if !ok || got.Name != "b" {
		t.Errorf("selected() = %q, %v, want \"b\", true", got.Name, ok)
	}
}

func TestListWindowFollowsCursor(t *testing.T) {
	t.Parallel()

	l := list{height: 3}
	l.setItems(items(6))

	tests := []struct {
		name      string
		cursor    int
		want      []string
		wantFirst int
	}{
		{name: "top", cursor: 0, want: []string{"a", "b", "c"}, wantFirst: 0},
		// Moving within the window must not scroll.
		{name: "inside window", cursor: 2, want: []string{"a", "b", "c"}, wantFirst: 0},
		{name: "past window", cursor: 3, want: []string{"b", "c", "d"}, wantFirst: 1},
		{name: "bottom", cursor: 5, want: []string{"d", "e", "f"}, wantFirst: 3},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			l.setCursor(test.cursor)
			got, first := l.window()
			if !slices.Equal(names(got), test.want) {
				t.Errorf("window() = %v, want %v", names(got), test.want)
			}
			if first != test.wantFirst {
				t.Errorf("window() first = %d, want %d", first, test.wantFirst)
			}
		})
	}
}

func TestListSetHeightKeepsCursorVisible(t *testing.T) {
	t.Parallel()

	l := list{height: 6}
	l.setItems(items(6))
	l.setCursor(5)

	l.setHeight(2)

	got, first := l.window()
	if !slices.Equal(names(got), []string{"e", "f"}) || first != 4 {
		t.Errorf("window() = %v, %d, want [e f], 4", names(got), first)
	}
}
