package ui

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/Br0ce/boltcutter/tree"
	"github.com/Br0ce/boltcutter/value"
)

// testDB is the database the tests say they are browsing.
var testDB = DB{Path: "testdata/test.db", Size: 4 * 1024 * 1024}

// newTestModel returns a model sized to a terminal large enough that
// nothing scrolls, and closes it with the test.
func newTestModel(t *testing.T, ft *fakeTree) Model {
	t.Helper()

	m, err := New(ft, testDB)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	sized, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}
	t.Cleanup(sized.Close)

	return sized
}

// press sends a key to the model and returns the updated model.
func press(t *testing.T, m Model, key string) Model {
	t.Helper()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	return next
}

// pressType sends a non-rune key, such as enter or tab.
func pressType(t *testing.T, m Model, key tea.KeyType) Model {
	t.Helper()

	updated, _ := m.Update(tea.KeyMsg{Type: key})
	next, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	return next
}

// selectEntry moves the cursor of the current column onto the entry
// named name.
func selectEntry(t *testing.T, m Model, name string) Model {
	t.Helper()

	for range len(m.current().items) {
		if entry, ok := m.current().selected(); ok && entry.Name == name {
			return m
		}
		m = press(t, m, "j")
	}
	t.Fatalf("no entry %q in the current column, got %v", name, names(m.current().items))

	return m
}

// texts renders the lines of a value as plain text, one string per line.
func texts(lines []value.Line) []string {
	out := make([]string, 0, len(lines))
	for _, line := range lines {
		out = append(out, line.Text())
	}

	return out
}

func TestNewOpensRoot(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())

	if len(m.columns) != 1 {
		t.Fatalf("columns = %d, want 1", len(m.columns))
	}
	if got := names(m.current().items); !slices.Equal(got, []string{"blobs", "config", "users"}) {
		t.Errorf("root listing = %v, want the three buckets", got)
	}
	if len(m.path()) != 0 {
		t.Errorf("path at the root = %q, want it empty", m.path())
	}
}

func TestPreviewShowsTheListingOfANode(t *testing.T) {
	t.Parallel()

	// The cursor starts on "blobs", so the preview column is already
	// holding the listing that descending into it would show.
	m := newTestModel(t, newFakeTree())

	if m.preview.column == nil {
		t.Fatal("no listing in the preview, want the one under the cursor")
	}
	if got := names(m.preview.column.items); !slices.Equal(got, []string{"huge", "raw"}) {
		t.Errorf("preview listing = %v, want the entries of blobs", got)
	}
	if m.preview.lines != nil {
		t.Errorf("preview holds value lines as well as a listing: %v", texts(m.preview.lines))
	}
}

func TestDescendTakesThePreviewColumn(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "config")
	previewed := m.preview.column

	m = pressType(t, m, tea.KeyEnter)

	if len(m.columns) != 2 {
		t.Fatalf("columns after descending = %d, want 2", len(m.columns))
	}
	// The listing was already open in the preview; descending moves
	// it across rather than reading it again.
	if m.current() != previewed {
		t.Error("descending opened a new listing, want the one the preview held")
	}
	if want := (tree.Path{"config"}); !slices.Equal(m.path(), want) {
		t.Errorf("path = %q, want %q", m.path(), want)
	}
	if got := names(m.current().items); !slices.Equal(got, []string{"flags", "version"}) {
		t.Errorf("current listing = %v, want the entries of config", got)
	}
}

func TestDescendIntoALeafIsANoop(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "users")
	m = pressType(t, m, tea.KeyEnter)
	m = selectEntry(t, m, "user:001")

	before := len(m.columns)
	m = pressType(t, m, tea.KeyEnter)

	if len(m.columns) != before {
		t.Errorf("columns after opening a key = %d, want %d", len(m.columns), before)
	}
	if entry, _ := m.current().selected(); entry.Name != "user:001" {
		t.Errorf("selection after opening a key = %q, want it unmoved", entry.Name)
	}
}

func TestAscendReturnsToTheEntryWeCameThrough(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "users")
	m = pressType(t, m, tea.KeyEnter)
	m = pressType(t, m, tea.KeyEsc)

	if len(m.columns) != 1 {
		t.Fatalf("columns after going back = %d, want 1", len(m.columns))
	}
	entry, ok := m.current().selected()
	if !ok || entry.Name != "users" {
		t.Errorf("selection after going back = %q, want \"users\"", entry.Name)
	}
	// The preview follows the cursor back out.
	if m.preview.column == nil {
		t.Fatal("no listing in the preview after going back")
	}
	if got := names(m.preview.column.items); !slices.Equal(got, []string{"user:001", "user:002"}) {
		t.Errorf("preview after going back = %v, want the entries of users", got)
	}
}

func TestAscendAtTheRootIsANoop(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	m = pressType(t, m, tea.KeyEsc)

	if len(m.columns) != 1 {
		t.Errorf("columns = %d, want 1", len(m.columns))
	}
}

func TestPreviewShowsAJSONValue(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "users")
	m = pressType(t, m, tea.KeyEnter)
	m = selectEntry(t, m, "user:002")

	if m.preview.column != nil {
		t.Error("preview holds a listing, want a value")
	}
	if got := texts(m.preview.lines); !slices.Equal(got, []string{"{", `  "id": 2`, "}"}) {
		t.Errorf("preview = %q, want the value rendered", got)
	}
}

func TestPreviewNotesAValueItCannotRead(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "blobs")
	m = pressType(t, m, tea.KeyEnter)
	m = selectEntry(t, m, "raw")

	if m.preview.lines != nil {
		t.Errorf("preview = %q, want no lines for a value we cannot read", texts(m.preview.lines))
	}
	if !strings.Contains(m.preview.note, "no format") {
		t.Errorf("note = %q, want it to say the format is unreadable", m.preview.note)
	}
	if !strings.Contains(m.preview.note, "15 B") {
		t.Errorf("note = %q, want the size of the value in it", m.preview.note)
	}
}

func TestPreviewStopsAtTheReadLimit(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "blobs")
	m = pressType(t, m, tea.KeyEnter)
	m = selectEntry(t, m, "huge")

	// The column reads a page of a long value, not the whole of it, and
	// says so rather than pretending the format is at fault.
	if !strings.Contains(m.preview.note, "too long") {
		t.Errorf("note = %q, want it to say the value was cut short", m.preview.note)
	}
	if !strings.Contains(m.preview.note, "KiB") {
		t.Errorf("note = %q, want the size of the whole value in it", m.preview.note)
	}
}

func TestMovingTheCursorReloadsThePreview(t *testing.T) {
	t.Parallel()

	m := selectEntry(t, newTestModel(t, newFakeTree()), "users")
	m = pressType(t, m, tea.KeyEnter)

	if got := texts(m.preview.lines); !slices.Equal(got, []string{"{", `  "id": 1`, "}"}) {
		t.Fatalf("preview = %q, want the first value", got)
	}
	m = press(t, m, "j")
	if got := texts(m.preview.lines); !slices.Equal(got, []string{"{", `  "id": 2`, "}"}) {
		t.Errorf("preview after moving = %q, want the second value", got)
	}
}

func TestBrowsingClosesWhatItOpens(t *testing.T) {
	t.Parallel()

	ft := newFakeTree()
	m := newTestModel(t, ft)

	// Walk down and back out again, passing over every entry on the
	// way, so every listing the browser opens is one it has to close.
	m = selectEntry(t, m, "config")
	m = pressType(t, m, tea.KeyEnter)
	m = press(t, m, "j")
	m = pressType(t, m, tea.KeyEsc)
	m = selectEntry(t, m, "users")
	m = pressType(t, m, tea.KeyEnter)
	m = pressType(t, m, tea.KeyEsc)

	// One column for the root and one listing in the preview.
	if ft.open != 2 {
		t.Errorf("listings left open while browsing = %d, want 2", ft.open)
	}
	m.Close()
	if ft.open != 0 {
		t.Errorf("listings left open after Close = %d, want none", ft.open)
	}
}

func TestFocusScrollsAValue(t *testing.T) {
	t.Parallel()

	ft := newFakeTree()
	// A value taller than the column, which is what the focus is for.
	ft.nodes["tall"] = []tree.Entry{{Name: "value", Kind: tree.Leaf}}
	ft.nodes[""] = append(ft.nodes[""], tree.Entry{Name: "tall", Kind: tree.Node})
	ft.values["tall/value"] = []byte(
		`{"a":1,"b":2,"c":3,"d":4,"e":5,"f":6,"g":7,"h":8,"i":9,"j":10,"k":11,"l":12}`)

	m := newTestModel(t, ft)
	m = selectEntry(t, m, "tall")
	m = pressType(t, m, tea.KeyEnter)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 12})
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	m = pressType(t, m, tea.KeyTab)
	if m.focus != focusValue {
		t.Fatalf("focus = %v, want the value", m.focus)
	}
	m = press(t, m, "j")
	if m.preview.offset != 1 {
		t.Errorf("offset after scrolling = %d, want 1", m.preview.offset)
	}
	// The cursor of the listing did not move with it.
	if entry, _ := m.current().selected(); entry.Name != "value" {
		t.Errorf("selection = %q, want it unmoved", entry.Name)
	}

	m = pressType(t, m, tea.KeyTab)
	if m.focus != focusListing {
		t.Errorf("focus = %v, want the listing", m.focus)
	}
}

func TestFocusStaysOnTheListingWithoutAValue(t *testing.T) {
	t.Parallel()

	// The cursor is on a bucket, so there is nothing in the preview to
	// scroll and the focus has nowhere to go.
	m := newTestModel(t, newFakeTree())
	m = pressType(t, m, tea.KeyTab)

	if m.focus != focusListing {
		t.Errorf("focus = %v, want the listing", m.focus)
	}
}

func TestErrorIsShownNotFatal(t *testing.T) {
	t.Parallel()

	ft := newFakeTree()
	broken := errors.New("bucket is on fire")
	ft.fail(tree.Path{"blobs"}, broken)

	m := newTestModel(t, ft)

	// The cursor starts on blobs, whose preview cannot be opened.
	if !errors.Is(m.err, broken) {
		t.Errorf("err = %v, want the failure of the preview", m.err)
	}
	if !strings.Contains(ansi.Strip(m.View()), "on fire") {
		t.Error("the view does not show the error")
	}
	// Moving off it clears the error and the browser carries on.
	m = press(t, m, "j")
	if m.err != nil {
		t.Errorf("err after moving on = %v, want none", m.err)
	}
}

func TestNewFailsOnARootThatWillNotOpen(t *testing.T) {
	t.Parallel()

	ft := newFakeTree()
	ft.fail(nil, errors.New("no such database"))

	if _, err := New(ft, testDB); err == nil {
		t.Error("New over a tree with no root returned no error")
	}
}

func TestUnprintableNamesAreMadeSafe(t *testing.T) {
	t.Parallel()

	// A bbolt key is any byte string, so a listing may hold one that
	// would wreck the screen if it were written out as it is.
	ft := newFakeTree()
	ft.nodes[""] = []tree.Entry{{Name: "a\nb\x1b[31m\xff", Kind: tree.Leaf}}
	ft.values["a\nb\x1b[31m\xff"] = []byte("1")

	view := ansi.Strip(newTestModel(t, ft).View())

	if strings.Contains(view, "\x1b[31m") {
		t.Error("the view carries an escape sequence out of a key")
	}
	if lines := strings.Split(view, "\n"); len(lines) != 24 {
		t.Errorf("view height = %d lines, want 24: a newline in a key broke the layout", len(lines))
	}
}

func TestQuit(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("q")})
	if cmd == nil {
		t.Fatal("q returned no command, want tea.Quit")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("q returned %T, want tea.QuitMsg", cmd())
	}
}

func TestViewLayout(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	view := m.View()

	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Errorf("view height = %d lines, want 24", len(lines))
	}
	for i, line := range lines {
		if got := lipgloss.Width(line); got != 100 {
			t.Errorf("line %d width = %d, want 100", i, got)
		}
	}
	// The header names the open bucket, the file and its size, the
	// footer the shortcuts, and the columns list the root buckets,
	// marked with a trailing slash.
	for _, want := range []string{bucketLabel, dbLabel, testDB.Path, "4.0 MiB", "config/", "users/", "quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
}

func TestColumnsFillFromTheLeft(t *testing.T) {
	t.Parallel()

	// At the root there is no parent, so the listing takes the
	// leftmost column, its preview sits right beside it, and the one
	// left empty is the rightmost.
	m := newTestModel(t, newFakeTree())

	root := []string{"› blobs/", "config/"}
	if got := columnLines(t, m.View(), columnLeft); !slices.Equal(got, root) {
		t.Errorf("left column at the root = %q, want the root listing %q", got, root)
	}
	if got := columnLines(t, m.View(), columnMiddle); !slices.Equal(got, []string{"huge", "raw"}) {
		t.Errorf("middle column at the root = %q, want the preview of blobs", got)
	}
	if got := columnLines(t, m.View(), columnRight); !slices.Equal(got, []string{"", ""}) {
		t.Errorf("right column at the root = %q, want it empty", got)
	}
}

func TestColumnsShiftAlongWhenDescending(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	m = selectEntry(t, m, "config")
	m = pressType(t, m, tea.KeyEnter)

	// One level down every column is in use: the parent we came through,
	// the listing we are in, and the preview of what the cursor is on.
	if got := columnLines(t, m.View(), columnLeft); !slices.Equal(got, []string{"blobs/", "› config/"}) {
		t.Errorf("left column = %q, want the root listing", got)
	}
	if got := columnLines(t, m.View(), columnMiddle); !slices.Equal(got, []string{"› flags/", "version"}) {
		t.Errorf("middle column = %q, want the entries of config", got)
	}
	if got := columnLines(t, m.View(), columnRight); !slices.Equal(got, []string{"beta", ""}) {
		t.Errorf("right column = %q, want the preview of flags", got)
	}

	// And back out again: the root listing returns to the leftmost
	// column rather than staying where it was pushed.
	m = pressType(t, m, tea.KeyEsc)
	if got := columnLines(t, m.View(), columnLeft); !slices.Equal(got, []string{"blobs/", "› config/"}) {
		t.Errorf("left column after going back = %q, want the root listing", got)
	}
	if got := columnLines(t, m.View(), columnRight); !slices.Equal(got, []string{"", ""}) {
		t.Errorf("right column after going back = %q, want it empty", got)
	}
}

func TestPreviewCarriesNoCursorMark(t *testing.T) {
	t.Parallel()

	// The preview is a listing nobody is in, so it carries no cursor
	// mark: the mark belongs to the column the keys are moving.
	m := newTestModel(t, newFakeTree())

	if got := columnLines(t, m.View(), columnMiddle); !slices.Equal(got, []string{"huge", "raw"}) {
		t.Errorf("preview column = %q, want the entries of blobs, unmarked", got)
	}
}

// columnLines returns the first two content lines of one column of a
// rendered view, without its border, padding or styling.
func columnLines(t *testing.T, view string, column int) []string {
	t.Helper()

	widths := [columnCount]int{25, 25, 50}

	start := 0
	for i := columnLeft; i < column; i++ {
		start += widths[i]
	}

	var out []string
	// Skip the header and the top border of the columns.
	for _, line := range strings.Split(view, "\n")[headerHeight+1:][:2] {
		runes := []rune(ansi.Strip(line))
		out = append(out, strings.TrimSpace(string(runes[start+2:start+widths[column]-2])))
	}

	return out
}

func TestLayoutIsFixed(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())

	// Three columns at the root, and the same three, at the same widths,
	// after diving: the columns shift but nothing resizes.
	root := measureColumns(t, m.View())
	if len(root) != 3 {
		t.Fatalf("columns at the root = %d, want 3", len(root))
	}

	m = pressType(t, m, tea.KeyEnter)

	if dived := measureColumns(t, m.View()); !slices.Equal(dived, root) {
		t.Errorf("columns after diving = %v, want %v", dived, root)
	}
}

// measureColumns measures the columns of a rendered view by splitting
// their top border line at the corners.
func measureColumns(t *testing.T, view string) []int {
	t.Helper()

	// The header sits above the columns.
	return measureBoxes(t, strings.Split(view, "\n")[headerHeight])
}

// measureBoxes measures a row of boxes by splitting the border line
// they start with at its corners.
func measureBoxes(t *testing.T, border string) []int {
	t.Helper()

	var widths []int
	for _, box := range strings.SplitAfter(border, "╮") {
		if box == "" {
			continue
		}
		widths = append(widths, lipgloss.Width(box))
	}

	return widths
}

func TestHeaderShowsWhereWeAreAndWhatWeAreIn(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	if got := headerText(m.View()); !strings.Contains(got, bucketLabel) {
		t.Errorf("header at the root = %q, want the bucket label in it", got)
	}

	m = selectEntry(t, m, "config")
	m = pressType(t, m, tea.KeyEnter)

	got := headerText(m.View())
	for _, want := range []string{"config", testDB.Path, "4.0 MiB"} {
		if !strings.Contains(got, want) {
			t.Errorf("header = %q, want %q in it", got, want)
		}
	}
	// The bucket leads: it is the fact that changes as we browse.
	if !strings.HasPrefix(strings.TrimSpace(got), bucketLabel+" ") {
		t.Errorf("header = %q, want it to open with the bucket", got)
	}
}

func TestHeaderFactsGiveWayInOrder(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	m = selectEntry(t, m, "config")
	m = pressType(t, m, tea.KeyEnter)
	m = selectEntry(t, m, "flags")
	m = pressType(t, m, tea.KeyEnter)

	// The bucket is the last fact to give way, and the size, being the
	// shortest, is kept long after the path it belongs to has gone.
	tests := map[string]struct {
		room         int
		bucket, file string
	}{
		"everything":     {room: 60, bucket: "at config/flags", file: "db testdata/test.db   ·   4.0 MiB"},
		"path shortened": {room: 50, bucket: "at config/flags", file: "db …tdata/test.db   ·   4.0 MiB"},
		"path dropped":   {room: 35, bucket: "at config/flags", file: "4.0 MiB"},
		"size dropped":   {room: 20, bucket: "at config/flags", file: ""},
		"bucket cut":     {room: 10, bucket: "at …/flags", file: ""},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			bucket, file := m.facts(test.room)
			if got := ansi.Strip(bucket); got != test.bucket {
				t.Errorf("facts(%d) bucket = %q, want %q", test.room, got, test.bucket)
			}
			if got := ansi.Strip(file); got != test.file {
				t.Errorf("facts(%d) file = %q, want %q", test.room, got, test.file)
			}
			if got := lipgloss.Width(bucket) + lipgloss.Width(file); got > test.room {
				t.Errorf("facts(%d) is %d cells wide", test.room, got)
			}
		})
	}
}

func TestHeaderShowsTheWholePathOfTheOpenBucket(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	m = selectEntry(t, m, "config")
	m = pressType(t, m, tea.KeyEnter)
	m = selectEntry(t, m, "flags")
	m = pressType(t, m, tea.KeyEnter)

	// Every bucket from the root down, while the bar has the room
	// for them.
	if got := headerText(m.View()); !strings.Contains(got, "config/flags") {
		t.Errorf("header = %q, want the whole path of the open bucket in it", got)
	}
}

func TestFooterShowsShortcuts(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	got := footerText(m.View())

	// A bar the width of the terminal is room for every shortcut at
	// its full wording.
	for _, s := range shortcuts {
		if want := s.spell(0).text(); !strings.Contains(got, want) {
			t.Errorf("footer = %q, want the %q shortcut in it", got, want)
		}
	}
}

func TestFooterShortensShortcutsToFitThemAllIn(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	narrow, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	// The full wording of them all does not fit a bar this narrow, so
	// every shortcut gives its words up together: a list spelled two
	// ways at once would read as two lists. A shortcut with nothing
	// shorter to say is spelled the one way and proves nothing here.
	got := footerText(narrow.View())
	for _, s := range shortcuts {
		full, brief := s.spell(0), s.spell(1)
		if !strings.Contains(got, brief.text()) {
			t.Errorf("footer = %q, want the %q shortcut in it", got, brief.text())
		}
		if brief != full && strings.Contains(got, full.text()) {
			t.Errorf("footer = %q, want %q shortened to %q", got, full.text(), brief.text())
		}
	}
}

func TestFooterFallsBackToBareKeys(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeTree())
	updated, _ := m.Update(tea.WindowSizeMsg{Width: 36, Height: 24})
	narrow, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	// Nothing fits beside its own words here, so the bar keeps the
	// keys and drops the words: a key is a reminder to whoever has
	// used the browser before, and the bar is the only place it is
	// written down.
	got := footerText(narrow.View())
	if !strings.Contains(got, shortcuts[0].keys) {
		t.Errorf("footer = %q, want the keys of the first shortcut in it", got)
	}
	for _, s := range shortcuts {
		if strings.Contains(got, s.help) {
			t.Errorf("footer = %q, want %q dropped at this width", got, s.help)
		}
	}
}

func TestHeaderTruncatesThePathFromTheLeft(t *testing.T) {
	t.Parallel()

	m, err := New(newFakeTree(), DB{Path: "/very/long/path/to/a/database/file.db", Size: 512})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(m.Close)

	updated, _ := m.Update(tea.WindowSizeMsg{Width: 56, Height: 24})
	narrow, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	// What is left of the path is its tail, which is the part that
	// names the file.
	got := headerText(narrow.View())
	if !strings.Contains(got, "file.db") {
		t.Errorf("header = %q, want the file name kept", got)
	}
	if !strings.Contains(got, ellipsis) {
		t.Errorf("header = %q, want the path truncated", got)
	}
	if strings.Contains(got, "/very/long") {
		t.Errorf("header = %q, want the head of the path dropped", got)
	}
}

func TestHeaderSizes(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		size int64
		want string
	}{
		"unknown":   {size: -1, want: unknownSize},
		"empty":     {size: 0, want: "0 B"},
		"bytes":     {size: 512, want: "512 B"},
		"kibibytes": {size: 4096, want: "4.0 KiB"},
		"mebibytes": {size: 3 * 1024 * 1024 / 2, want: "1.5 MiB"},
		"gibibytes": {size: 2 * 1024 * 1024 * 1024, want: "2.0 GiB"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			if got := formatSize(test.size); got != test.want {
				t.Errorf("formatSize(%d) = %q, want %q", test.size, got, test.want)
			}
		})
	}
}

// headerText returns the header bar of a rendered view, stripped of
// styling.
func headerText(view string) string {
	return ansi.Strip(strings.Split(view, "\n")[0])
}

// footerText returns the content line of the rendered footer, inside
// its border and stripped of styling.
func footerText(view string) string {
	lines := strings.Split(view, "\n")

	return ansi.Strip(lines[len(lines)-2])
}

func TestViewBeforeWindowSize(t *testing.T) {
	t.Parallel()

	m, err := New(newFakeTree(), testDB)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(m.Close)

	if got := m.View(); got != "" {
		t.Errorf("View before the first WindowSizeMsg = %q, want empty", got)
	}
}
