package ui

import (
	"errors"
	"slices"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Br0ce/boltcutter/store"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// fakeStore is an in-memory Store. Buckets are addressed by their path
// joined with "/", which is enough for tests that never use "/" in a
// bucket name.
type fakeStore struct {
	entries map[string][]store.Entry
	values  map[string]string
	err     error
}

// newFakeStore mirrors the shape of the golden fixture: a flat bucket of
// JSON values and a bucket holding both a key and a sub-bucket.
func newFakeStore() *fakeStore {
	return &fakeStore{
		entries: map[string][]store.Entry{
			"": {{Name: "config", Bucket: true}, {Name: "users", Bucket: true}},
			"config": {
				{Name: "flags", Bucket: true},
				{Name: "version"},
			},
			"config/flags": {{Name: "beta"}},
			"users":        {{Name: "user:001"}, {Name: "user:002"}},
		},
		values: map[string]string{
			"config/version":    `{"major":1}`,
			"config/flags/beta": `false`,
			"users/user:001":    `{"id":1}`,
			"users/user:002":    `not json`,
		},
	}
}

func (s *fakeStore) Entries(path []string) ([]store.Entry, error) {
	if s.err != nil {
		return nil, s.err
	}

	return s.entries[strings.Join(path, "/")], nil
}

func (s *fakeStore) Value(path []string, key string) ([]byte, error) {
	if s.err != nil {
		return nil, s.err
	}
	value, ok := s.values[strings.Join(append(append([]string{}, path...), key), "/")]
	if !ok {
		return nil, errors.New("key not found")
	}

	return []byte(value), nil
}

// press sends a key to the model and returns the updated model.
func press(t *testing.T, m Model, key string) Model {
	t.Helper()

	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	return got
}

// pressType sends a non-rune key, such as enter or tab.
func pressType(t *testing.T, m Model, key tea.KeyType) Model {
	t.Helper()

	updated, _ := m.Update(tea.KeyMsg{Type: key})
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	return got
}

// newTestModel returns a model sized to a terminal large enough that
// nothing scrolls.
func newTestModel(t *testing.T, store Store) Model {
	t.Helper()

	updated, _ := New(store, "testdata/test.db").Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	got, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	return got
}

// selectEntry moves the cursor of the current listing onto the entry
// named name.
func selectEntry(t *testing.T, m Model, name string) Model {
	t.Helper()

	for i, entry := range m.current().list.items {
		if entry.Name != name {
			continue
		}
		m.current().list.setCursor(i)
		m.loadValue()

		return m
	}
	t.Fatalf("no entry %q in %v", name, m.current().list.items)

	return m
}

func TestNewOpensRoot(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())

	if len(m.levels) != 1 {
		t.Fatalf("levels = %d, want 1", len(m.levels))
	}
	if got := names(m.current().list.items); !slices.Equal(got, []string{"config", "users"}) {
		t.Errorf("root listing = %v, want [config users]", got)
	}
	// The cursor starts on a bucket, and the preview is for values.
	if m.valueLines != nil {
		t.Errorf("valueLines = %q, want none with a bucket selected", m.valueLines)
	}
}

func TestDescendShiftsColumnsLeft(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	// The cursor starts on "config", which holds a sub-bucket.
	m = pressType(t, m, tea.KeyEnter)

	if !slices.Equal(m.path(), []string{"config"}) {
		t.Fatalf("path = %v, want [config]", m.path())
	}
	// What was previewed is now the current listing, and the root has
	// shifted into the parent column.
	if got := names(m.current().list.items); !slices.Equal(got, []string{"flags", "version"}) {
		t.Errorf("current listing = %v, want [flags version]", got)
	}
	parent := m.levels[len(m.levels)-2]
	if got := names(parent.list.items); !slices.Equal(got, []string{"config", "users"}) {
		t.Errorf("parent listing = %v, want [config users]", got)
	}

	// Diving into the nested bucket shifts once more.
	m = pressType(t, m, tea.KeyEnter)

	if !slices.Equal(m.path(), []string{"config", "flags"}) {
		t.Fatalf("path = %v, want [config flags]", m.path())
	}
	if got := names(m.current().list.items); !slices.Equal(got, []string{"beta"}) {
		t.Errorf("current listing = %v, want [beta]", got)
	}
}

func TestDescendIntoKeyIsANoop(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	m = pressType(t, m, tea.KeyEnter) // into config
	m = selectEntry(t, m, "version")

	m = pressType(t, m, tea.KeyEnter)

	if !slices.Equal(m.path(), []string{"config"}) {
		t.Errorf("path after entering a key = %v, want [config]", m.path())
	}
}

func TestAscendShiftsColumnsBack(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	m = pressType(t, m, tea.KeyEnter) // into config
	m = pressType(t, m, tea.KeyEnter) // into config/flags

	m = pressType(t, m, tea.KeyEsc)

	if !slices.Equal(m.path(), []string{"config"}) {
		t.Fatalf("path after esc = %v, want [config]", m.path())
	}
	// The cursor is back on the bucket that was left.
	entry, _ := m.current().list.selected()
	if entry.Name != "flags" {
		t.Errorf("selected = %q, want \"flags\"", entry.Name)
	}
}

func TestAscendAtRootIsANoop(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	m = pressType(t, m, tea.KeyEsc)

	if len(m.levels) != 1 {
		t.Errorf("levels = %d, want 1", len(m.levels))
	}
}

func TestPreviewShowsJSONValue(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	m = pressType(t, m, tea.KeyEnter) // into config
	m = selectEntry(t, m, "version")

	want := []string{"{", `  "major": 1`, "}"}
	if !slices.Equal(m.valueLines, want) {
		t.Errorf("valueLines = %q, want %q", m.valueLines, want)
	}
}

func TestPreviewIsEmptyForNonJSON(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	m = selectEntry(t, m, "users")
	m = pressType(t, m, tea.KeyEnter) // into users
	m = selectEntry(t, m, "user:002") // holds "not json"

	if m.valueLines != nil {
		t.Errorf("valueLines = %q, want none for a value that is not JSON", m.valueLines)
	}
	if m.err != nil {
		t.Errorf("err = %v, want nil: a non-JSON value is not an error", m.err)
	}
}

func TestMovingCursorReloadsValue(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	m = pressType(t, m, tea.KeyEnter) // into config, cursor on the flags bucket
	m = press(t, m, "j")              // down to the version key

	entry, _ := m.current().list.selected()
	if entry.Name != "version" {
		t.Fatalf("selected = %q, want \"version\"", entry.Name)
	}
	want := []string{"{", `  "major": 1`, "}"}
	if !slices.Equal(m.valueLines, want) {
		t.Errorf("valueLines = %q, want %q", m.valueLines, want)
	}

	// Moving back onto a bucket empties the pane again.
	m = press(t, m, "k")
	if m.valueLines != nil {
		t.Errorf("valueLines = %q, want none with a bucket selected", m.valueLines)
	}
}

func TestValuePaneScrolls(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	store.values["users/user:001"] = "[" + strings.Repeat("1,", 99) + "1]"
	m := newTestModel(t, store)
	m = selectEntry(t, m, "users")
	m = pressType(t, m, tea.KeyEnter) // into users, cursor on user:001

	m = pressType(t, m, tea.KeyTab)
	if m.focus != panePreview {
		t.Fatalf("focus = %v, want panePreview", m.focus)
	}
	m = press(t, m, "j")

	if m.valueOffset != 1 {
		t.Errorf("valueOffset = %d, want 1", m.valueOffset)
	}

	m = press(t, m, "G")
	if want := len(m.valueLines) - m.listHeight(); m.valueOffset != want {
		t.Errorf("valueOffset after G = %d, want %d", m.valueOffset, want)
	}

	// With the preview focused the listing cursor stays put.
	entry, _ := m.current().list.selected()
	if entry.Name != "user:001" {
		t.Errorf("selected = %q, want \"user:001\"", entry.Name)
	}
}

func TestReloadKeepsPosition(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	m := newTestModel(t, store)
	m = pressType(t, m, tea.KeyEnter) // into config
	m = selectEntry(t, m, "version")

	m = press(t, m, "r")

	if !slices.Equal(m.path(), []string{"config"}) {
		t.Fatalf("path after reload = %v, want [config]", m.path())
	}
	entry, _ := m.current().list.selected()
	if entry.Name != "version" {
		t.Errorf("selected after reload = %q, want \"version\"", entry.Name)
	}
}

func TestReloadDropsVanishedBucket(t *testing.T) {
	t.Parallel()

	fs := newFakeStore()
	m := newTestModel(t, fs)
	m = pressType(t, m, tea.KeyEnter) // into config
	m = pressType(t, m, tea.KeyEnter) // into config/flags

	// The sub-bucket disappears underneath the UI.
	fs.entries["config"] = []store.Entry{{Name: "version"}}
	m = press(t, m, "r")

	if !slices.Equal(m.path(), []string{"config"}) {
		t.Errorf("path after the bucket vanished = %v, want [config]", m.path())
	}
}

func TestStoreErrorIsShownNotFatal(t *testing.T) {
	t.Parallel()

	store := newFakeStore()
	m := newTestModel(t, store)

	store.err = errors.New("database gone")
	m = press(t, m, "r")

	if m.err == nil {
		t.Fatal("reload with a failing store left err nil")
	}
	if !strings.Contains(m.View(), "database gone") {
		t.Error("View does not show the store error")
	}
}

func TestQuit(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
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

	m := newTestModel(t, newFakeStore())
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
	// The header names the program, the open bucket and the file, the
	// panes list the root buckets — marked with a trailing slash — and the footer the
	// shortcuts.
	for _, want := range []string{"boltcutter", "bucket:", "db:", "testdata/test.db", "config/", "users/", "quit"} {
		if !strings.Contains(view, want) {
			t.Errorf("view does not contain %q", want)
		}
	}
}

func TestRootListingStaysInTheLeftPane(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	root := paneLines(t, m.View(), paneParent)
	if want := []string{"› config/", "users/"}; !slices.Equal(root, want) {
		t.Fatalf("left pane at the root = %q, want %q", root, want)
	}

	m = pressType(t, m, tea.KeyEnter) // into config
	m = pressType(t, m, tea.KeyEsc)   // and back out

	// Coming back to the root must not push the listing one pane to the
	// right; it stays where it was, cursor and all.
	if got := paneLines(t, m.View(), paneParent); !slices.Equal(got, root) {
		t.Errorf("left pane after going back = %q, want %q", got, root)
	}
	if got := paneLines(t, m.View(), paneCurrent); !slices.Equal(got, []string{"", ""}) {
		t.Errorf("middle pane after going back = %q, want it empty", got)
	}
}

// paneLines returns the first two content lines of one pane of a rendered
// view, without their border, padding or styling.
func paneLines(t *testing.T, view string, p pane) []string {
	t.Helper()

	widths := [paneCount]int{}
	copy(widths[:], []int{25, 25, 50})

	start := 0
	for i := paneParent; i < p; i++ {
		start += widths[i]
	}

	var out []string
	// Skip the header and the top border of the panes.
	for _, line := range strings.Split(view, "\n")[headerHeight+1:][:2] {
		runes := []rune(ansi.Strip(line))
		out = append(out, strings.TrimSpace(string(runes[start+2:start+widths[p]-2])))
	}

	return out
}

func TestLayoutIsFixed(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())

	// Three panes at the root, and the same three, at the same widths,
	// after diving: the columns shift but nothing resizes.
	root := columnWidths(t, m.View())
	if len(root) != 3 {
		t.Fatalf("columns at the root = %d, want 3", len(root))
	}

	m = pressType(t, m, tea.KeyEnter)

	if dived := columnWidths(t, m.View()); !slices.Equal(dived, root) {
		t.Errorf("columns after diving = %v, want %v", dived, root)
	}
}

// columnWidths measures the panes of a rendered view by splitting the top
// border line at the corners.
func columnWidths(t *testing.T, view string) []int {
	t.Helper()

	lines := strings.Split(view, "\n")
	top := lines[headerHeight] // the header sits above the panes.
	var widths []int
	for _, column := range strings.SplitAfter(top, "╮") {
		if column == "" {
			continue
		}
		widths = append(widths, lipgloss.Width(column))
	}

	return widths
}

func TestHeaderShowsPathOfOpenBucket(t *testing.T) {
	t.Parallel()

	m := newTestModel(t, newFakeStore())
	if got := header(m.View()); !strings.Contains(got, "/") {
		t.Errorf("header at the root = %q, want the root path in it", got)
	}

	m = pressType(t, m, tea.KeyEnter) // into config

	got := header(m.View())
	if !strings.Contains(got, "/config") {
		t.Errorf("header = %q, want the open bucket in it", got)
	}
	if !strings.Contains(got, "boltcutter") || !strings.Contains(got, "testdata/test.db") {
		t.Errorf("header = %q, want the program name and the database file in it", got)
	}
}

func TestHeaderDropsPathWhenNarrow(t *testing.T) {
	t.Parallel()

	updated, _ := New(newFakeStore(), "/very/long/path/to/a/database/file.db").
		Update(tea.WindowSizeMsg{Width: 40, Height: 24})
	m, ok := updated.(Model)
	if !ok {
		t.Fatalf("Update returned %T, want ui.Model", updated)
	}

	got := header(m.View())
	if lipgloss.Width(got) != 40 {
		t.Errorf("header width = %d, want 40", lipgloss.Width(got))
	}
	// The file is shortened to fit, the program name is kept whole.
	if !strings.Contains(got, "boltcutter") {
		t.Errorf("header = %q, want the program name kept", got)
	}
	if !strings.Contains(got, "…") {
		t.Errorf("header = %q, want the database file truncated", got)
	}
}

// header returns the content line of the rendered header, inside its
// border.
func header(view string) string {
	return strings.Split(view, "\n")[1]
}

func TestViewBeforeWindowSize(t *testing.T) {
	t.Parallel()

	if got := New(newFakeStore(), "testdata/test.db").View(); got != "" {
		t.Errorf("View before the first WindowSizeMsg = %q, want empty", got)
	}
}
