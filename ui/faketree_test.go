package ui

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/Br0ce/boltcutter/tree"
)

// fakeTree is an in-memory tree. Nodes and leaves are addressed by
// their path joined with "/", which is enough for tests that never use
// "/" in a name.
//
// It counts the listings it has handed out and not had back, so a test
// can tell whether the browser closes what it opens.
type fakeTree struct {
	nodes  map[string][]tree.Entry
	values map[string][]byte
	errs   map[string]error

	open int
}

func key(path tree.Path) string {
	return strings.Join(path, "/")
}

// newFakeTree mirrors the shape of a small database: a bucket of JSON
// values, a bucket holding both a key and a sub-bucket, and a bucket of
// values that are no JSON at all.
func newFakeTree() *fakeTree {
	node := func(name string) tree.Entry { return tree.Entry{Name: name, Kind: tree.Node} }
	leaf := func(name string) tree.Entry { return tree.Entry{Name: name, Kind: tree.Leaf} }

	return &fakeTree{
		nodes: map[string][]tree.Entry{
			"":             {node("blobs"), node("config"), node("users")},
			"blobs":        {leaf("huge"), leaf("raw")},
			"config":       {node("flags"), leaf("version")},
			"config/flags": {leaf("beta")},
			"users":        {leaf("user:001"), leaf("user:002")},
		},
		values: map[string][]byte{
			"blobs/huge":        []byte(strings.Repeat("x", 3*previewLimit)),
			"blobs/raw":         []byte("not json at all"),
			"config/version":    []byte(`"1"`),
			"config/flags/beta": []byte("false"),
			"users/user:001":    []byte(`{"id":1}`),
			"users/user:002":    []byte(`{"id":2}`),
		},
		errs: map[string]error{},
	}
}

// withSequence adds a bucket of n leaves named k00, k01, … for tests
// that need more entries than a column can hold.
func (t *fakeTree) withSequence(name string, n int) *fakeTree {
	entries := make([]tree.Entry, 0, n)
	for i := range n {
		leaf := tree.Entry{Name: seqName(i), Kind: tree.Leaf}
		entries = append(entries, leaf)
		t.values[name+"/"+leaf.Name] = []byte(leaf.Name)
	}
	t.nodes[name] = entries
	t.nodes[""] = append(t.nodes[""], tree.Entry{Name: name, Kind: tree.Node})
	slices.SortFunc(t.nodes[""], func(a, b tree.Entry) int { return strings.Compare(a.Name, b.Name) })

	return t
}

func seqName(i int) string {
	return fmt.Sprintf("k%02d", i)
}

// fail makes the path fail to open or read, so a test can see what the
// browser does with an error.
func (t *fakeTree) fail(path tree.Path, err error) {
	t.errs[key(path)] = err
}

func (t *fakeTree) Open(path tree.Path) (tree.Listing, error) {
	if err := t.errs[key(path)]; err != nil {
		return nil, err
	}
	entries, ok := t.nodes[key(path)]
	if !ok {
		return nil, errNotFound
	}
	t.open++

	return &fakeListing{tree: t, entries: entries, at: -1}, nil
}

func (t *fakeTree) Read(path tree.Path, limit int) ([]byte, int, error) {
	if err := t.errs[key(path)]; err != nil {
		return nil, 0, err
	}
	value, ok := t.values[key(path)]
	if !ok {
		return nil, 0, errNotFound
	}
	size := len(value)
	if limit >= 0 && limit < size {
		value = value[:limit]
	}

	return slices.Clone(value), size, nil
}

// fakeListing walks a slice the way store's listing walks a bbolt
// cursor, and holds itself to the same contract: it starts unplaced,
// and a move that runs off an end keeps the place it had.
type fakeListing struct {
	tree    *fakeTree
	entries []tree.Entry
	at      int
	closed  bool
}

func (l *fakeListing) First() (tree.Entry, bool) {
	return l.placeAt(0)
}

func (l *fakeListing) Last() (tree.Entry, bool) {
	return l.placeAt(len(l.entries) - 1)
}

func (l *fakeListing) Next() (tree.Entry, bool) {
	if l.at < 0 {
		return tree.Entry{}, false
	}

	return l.placeAt(l.at + 1)
}

func (l *fakeListing) Prev() (tree.Entry, bool) {
	if l.at < 0 {
		return tree.Entry{}, false
	}

	return l.placeAt(l.at - 1)
}

func (l *fakeListing) Seek(name string) (tree.Entry, bool) {
	for i, entry := range l.entries {
		if entry.Name >= name {
			return l.placeAt(i)
		}
	}

	return l.placeAt(len(l.entries))
}

func (l *fakeListing) placeAt(i int) (tree.Entry, bool) {
	if l.closed || i < 0 || i >= len(l.entries) {
		return tree.Entry{}, false
	}
	l.at = i

	return l.entries[i], true
}

func (l *fakeListing) Close() error {
	if !l.closed {
		l.closed = true
		l.tree.open--
	}

	return nil
}

// names lists the entries of a window by name.
func names(entries []tree.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}

	return out
}

// openColumn opens a column over a fake tree, and closes it with the
// test.
func openColumn(t *testing.T, ft *fakeTree, path tree.Path, height int) *column {
	t.Helper()

	c, err := newColumn(ft, path, height)
	if err != nil {
		t.Fatalf("newColumn %q: %v", path, err)
	}
	t.Cleanup(func() {
		if err := c.Close(); err != nil {
			t.Errorf("close column: %v", err)
		}
	})

	return c
}

var (
	_ tree.Tree    = (*fakeTree)(nil)
	_ tree.Listing = (*fakeListing)(nil)
)
