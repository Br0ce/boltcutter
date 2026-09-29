package store

import (
	"errors"
	"slices"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/Br0ce/boltcutter/internal/testutil"
	"github.com/Br0ce/boltcutter/tree"
)

// newTestDB returns a throwaway database holding a flat bucket, a
// bucket with a nested sub-bucket, and a bucket of values worth
// reading carefully: an empty one and one too long to show whole.
func newTestDB(t *testing.T) *bolt.DB {
	t.Helper()

	return testutil.ReadOnlyDB(t, func(tx *bolt.Tx) error {
		users, err := tx.CreateBucket([]byte("users"))
		if err != nil {
			return err
		}
		if err := users.Put([]byte("user:001"), []byte(`{"id":1}`)); err != nil {
			return err
		}

		config, err := tx.CreateBucket([]byte("config"))
		if err != nil {
			return err
		}
		if err := config.Put([]byte("version"), []byte("1")); err != nil {
			return err
		}
		flags, err := config.CreateBucket([]byte("flags"))
		if err != nil {
			return err
		}
		if err := flags.Put([]byte("beta"), []byte("false")); err != nil {
			return err
		}

		blobs, err := tx.CreateBucket([]byte("blobs"))
		if err != nil {
			return err
		}
		if err := blobs.Put([]byte("empty"), []byte{}); err != nil {
			return err
		}

		return blobs.Put([]byte("long"), []byte("0123456789"))
	})
}

// newTestStore returns a Bolt over a throwaway database. The Bolt is
// closed before the database is, because a database cannot be closed
// while a transaction is still reading it.
func newTestStore(t *testing.T) *Bolt {
	t.Helper()

	return storeOver(t, newTestDB(t))
}

func storeOver(t *testing.T, db *bolt.DB) *Bolt {
	t.Helper()

	store, err := New(db)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	})

	return store
}

// walk lists a path from its first entry to its last.
func walk(t *testing.T, store *Bolt, path tree.Path) []tree.Entry {
	t.Helper()

	listing, err := store.Open(path)
	if err != nil {
		t.Fatalf("Open %q: %v", path, err)
	}
	defer listing.Close()

	var entries []tree.Entry
	for entry, ok := listing.First(); ok; entry, ok = listing.Next() {
		entries = append(entries, entry)
	}

	return entries
}

// names lists the entries by name.
func names(entries []tree.Entry) []string {
	out := make([]string, 0, len(entries))
	for _, entry := range entries {
		out = append(out, entry.Name)
	}

	return out
}

func TestRootHoldsBucketsOnly(t *testing.T) {
	t.Parallel()

	entries := walk(t, newTestStore(t), nil)

	if got := names(entries); !slices.Equal(got, []string{"blobs", "config", "users"}) {
		t.Fatalf("root entries = %v, want [blobs config users]", got)
	}
	for _, entry := range entries {
		if entry.Kind != tree.Node {
			t.Errorf("root entry %q is a %v, want a node", entry.Name, entry.Kind)
		}
	}
}

func TestListingTellsNodesFromLeaves(t *testing.T) {
	t.Parallel()

	// config holds a sub-bucket and a key, listed together in key
	// order.
	entries := walk(t, newTestStore(t), tree.Path{"config"})

	if got := names(entries); !slices.Equal(got, []string{"flags", "version"}) {
		t.Fatalf("config entries = %v, want [flags version]", got)
	}
	if entries[0].Kind != tree.Node {
		t.Errorf("flags is a %v, want a node", entries[0].Kind)
	}
	if entries[1].Kind != tree.Leaf {
		t.Errorf("version is a %v, want a leaf", entries[1].Kind)
	}
}

func TestEmptyValueIsALeaf(t *testing.T) {
	t.Parallel()

	// bbolt marks a nested bucket with a nil value and an empty value
	// with an empty slice. Blur the two and every empty key reads as
	// a bucket.
	entries := walk(t, newTestStore(t), tree.Path{"blobs"})

	if got := names(entries); !slices.Equal(got, []string{"empty", "long"}) {
		t.Fatalf("blobs entries = %v, want [empty long]", got)
	}
	if entries[0].Kind != tree.Leaf {
		t.Errorf("the empty value is a %v, want a leaf", entries[0].Kind)
	}
}

func TestEmptyBucketIsNotAnError(t *testing.T) {
	t.Parallel()

	db := testutil.ReadOnlyDB(t, func(tx *bolt.Tx) error {
		_, err := tx.CreateBucket([]byte("empty"))

		return err
	})

	listing, err := storeOver(t, db).Open(tree.Path{"empty"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer listing.Close()

	if entry, ok := listing.First(); ok {
		t.Errorf("first entry of an empty bucket = %q, want none", entry.Name)
	}
	if _, ok := listing.Last(); ok {
		t.Errorf("last entry of an empty bucket exists, want none")
	}
}

func TestOpenWhatCannotBeOpened(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	tests := []struct {
		name string
		path tree.Path
	}{
		{"a bucket that is not there", tree.Path{"nope"}},
		{"a bucket under a bucket that is not there", tree.Path{"nope", "deeper"}},
		{"a sub-bucket that is not there", tree.Path{"config", "nope"}},
		{"a key", tree.Path{"config", "version"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, err := store.Open(test.path); !errors.Is(err, ErrNotFound) {
				t.Errorf("Open %q = %v, want ErrNotFound", test.path, err)
			}
		})
	}
}

func TestReadReportsTheWholeSize(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	path := tree.Path{"config", "flags", "beta"}

	data, size, err := store.Read(path, -1)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if string(data) != "false" {
		t.Errorf("Read = %q, want \"false\"", data)
	}
	if size != len("false") {
		t.Errorf("size = %d, want %d", size, len("false"))
	}
}

func TestReadStopsAtTheLimit(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)
	path := tree.Path{"blobs", "long"}

	tests := []struct {
		name  string
		limit int
		want  string
	}{
		{"a limit under the size", 4, "0123"},
		{"a limit at the size", 10, "0123456789"},
		{"a limit over the size", 99, "0123456789"},
		{"no limit", -1, "0123456789"},
		{"a limit of nothing", 0, ""},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			data, size, err := store.Read(path, test.limit)
			if err != nil {
				t.Fatalf("Read: %v", err)
			}
			if string(data) != test.want {
				t.Errorf("Read = %q, want %q", data, test.want)
			}
			// The size is the value's, never the limit's, so a
			// reader can tell it is holding a prefix.
			if size != 10 {
				t.Errorf("size = %d, want 10", size)
			}
		})
	}
}

func TestReadWhatCannotBeRead(t *testing.T) {
	t.Parallel()

	store := newTestStore(t)

	tests := []struct {
		name string
		path tree.Path
	}{
		{"a key that is not there", tree.Path{"users", "nope"}},
		{"a bucket", tree.Path{"config", "flags"}},
		{"a top-level bucket", tree.Path{"config"}},
		{"the root", nil},
		{"a key in a bucket that is not there", tree.Path{"nope", "key"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			if _, _, err := store.Read(test.path, -1); !errors.Is(err, ErrNotFound) {
				t.Errorf("Read %q = %v, want ErrNotFound", test.path, err)
			}
		})
	}
}

func TestListingStartsUnplaced(t *testing.T) {
	t.Parallel()

	listing, err := newTestStore(t).Open(tree.Path{"blobs"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer listing.Close()

	if entry, ok := listing.Next(); ok {
		t.Errorf("Next before placing = %q, want nothing", entry.Name)
	}
	if entry, ok := listing.Prev(); ok {
		t.Errorf("Prev before placing = %q, want nothing", entry.Name)
	}
	// Placing it works after that: the failed moves left no mess.
	if entry, ok := listing.First(); !ok || entry.Name != "empty" {
		t.Errorf("First = %q, %v, want \"empty\", true", entry.Name, ok)
	}
}

func TestListingHoldsItsPlaceAtTheEnds(t *testing.T) {
	t.Parallel()

	listing, err := newTestStore(t).Open(tree.Path{"blobs"})
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer listing.Close()

	// Walk into the last entry, then off the end.
	listing.First()
	if entry, ok := listing.Next(); !ok || entry.Name != "long" {
		t.Fatalf("Next = %q, %v, want \"long\", true", entry.Name, ok)
	}
	if entry, ok := listing.Next(); ok {
		t.Errorf("Next past the end = %q, want nothing", entry.Name)
	}
	// The failed move left the listing on "long", so backing out of
	// the end works.
	if entry, ok := listing.Prev(); !ok || entry.Name != "empty" {
		t.Errorf("Prev after running off the end = %q, %v, want \"empty\", true", entry.Name, ok)
	}

	// The same at the other end.
	if entry, ok := listing.Prev(); ok {
		t.Errorf("Prev past the start = %q, want nothing", entry.Name)
	}
	if entry, ok := listing.Next(); !ok || entry.Name != "long" {
		t.Errorf("Next after running off the start = %q, %v, want \"long\", true", entry.Name, ok)
	}
}

func TestListingLastWalksBackwards(t *testing.T) {
	t.Parallel()

	listing, err := newTestStore(t).Open(nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer listing.Close()

	var got []string
	for entry, ok := listing.Last(); ok; entry, ok = listing.Prev() {
		got = append(got, entry.Name)
	}
	if !slices.Equal(got, []string{"users", "config", "blobs"}) {
		t.Errorf("walking backwards = %v, want [users config blobs]", got)
	}
}

func TestListingSeek(t *testing.T) {
	t.Parallel()

	listing, err := newTestStore(t).Open(nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer listing.Close()

	if entry, ok := listing.Seek("config"); !ok || entry.Name != "config" {
		t.Errorf("Seek to a name that is there = %q, %v, want \"config\", true", entry.Name, ok)
	}
	// A name that is not there lands on the next one along.
	if entry, ok := listing.Seek("d"); !ok || entry.Name != "users" {
		t.Errorf("Seek past a name = %q, %v, want \"users\", true", entry.Name, ok)
	}
	// A name past every entry finds nothing and holds its place.
	if entry, ok := listing.Seek("zzz"); ok {
		t.Errorf("Seek past the end = %q, want nothing", entry.Name)
	}
	if entry, ok := listing.Prev(); !ok || entry.Name != "config" {
		t.Errorf("Prev after a seek past the end = %q, %v, want \"config\", true", entry.Name, ok)
	}
}

func TestClosedListingMovesNowhere(t *testing.T) {
	t.Parallel()

	listing, err := newTestStore(t).Open(nil)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if _, ok := listing.First(); !ok {
		t.Fatalf("First before closing found nothing")
	}
	if err := listing.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	// Closing twice is allowed and does nothing.
	if err := listing.Close(); err != nil {
		t.Fatalf("Close twice: %v", err)
	}

	for _, move := range []struct {
		name string
		move func() (tree.Entry, bool)
	}{
		{"First", listing.First},
		{"Last", listing.Last},
		{"Next", listing.Next},
		{"Prev", listing.Prev},
		{"Seek", func() (tree.Entry, bool) { return listing.Seek("config") }},
	} {
		if entry, ok := move.move(); ok {
			t.Errorf("%s on a closed listing = %q, want nothing", move.name, entry.Name)
		}
	}
}

func TestNewRefusesAWritableDatabase(t *testing.T) {
	t.Parallel()

	// A Bolt holds a read transaction for as long as it lives, and a
	// write that has to remap the file waits for every open read
	// transaction. A writable database is therefore not a store that
	// works a little worse, it is one that hangs whoever writes.
	db := testutil.TempDB(t)

	store, err := New(db)
	if !errors.Is(err, ErrWritable) {
		t.Errorf("New over a writable database = %v, want ErrWritable", err)
	}
	if store != nil {
		t.Errorf("New returned a store alongside the error")
		if err := store.Close(); err != nil {
			t.Errorf("close store: %v", err)
		}
	}
}
