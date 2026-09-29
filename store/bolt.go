// Package store reads a bbolt database as a tree of nodes and leaves.
// Buckets are the nodes and key/value pairs the leaves.
//
// # One transaction, one snapshot
//
// A Bolt holds a single read-only transaction open for as long as it
// lives. That is what lets a listing be a cursor into the database
// rather than a copy of it: the pages it walks stay where they are,
// and a bucket with a million keys costs a pane of forty lines
// nothing. It is also what the tree package means by a snapshot —
// everything a Bolt says comes from the one state of the file it
// opened, and it will not notice a later one.
//
// The price is steep if anything is writing: a write that has to grow
// the file remaps it, and remapping waits for every open read
// transaction, so a writer does not merely make the file grow — it
// stalls for as long as a Bolt is browsing. New therefore refuses a
// database that was not opened read-only, which takes a shared lock
// and keeps writers out of the file entirely. The price is then not
// paid at all, and a writer that tries is turned away by the lock
// instead of hanging on it.
//
// Nothing here interprets a value: a leaf hands over the bytes as they
// were stored. Keys and values live in the memory-mapped file for the
// life of the transaction, so everything that leaves the package is
// copied out first, and a Bolt outliving its caller cannot hand back
// memory that has been unmapped.
//
// A Bolt is not safe for concurrent use: a read transaction is walked
// by one goroutine at a time.
package store

import (
	"errors"
	"fmt"

	bolt "go.etcd.io/bbolt"

	"github.com/Br0ce/boltcutter/tree"
)

// ErrNotFound is returned when a path names a bucket or a key the
// database does not hold. Callers match on it with errors.Is.
var ErrNotFound = errors.New("not found")

// ErrWritable is returned by New when the database was not opened
// read-only. See the package comment: a Bolt holds a read transaction
// open, and a writer sharing the file with one stalls behind it.
var ErrWritable = errors.New("database is not open read-only")

// Bolt reads a bbolt database at the state it had when it was opened.
type Bolt struct {
	tx *bolt.Tx
}

// New returns a Bolt reading from db, and opens the read-only
// transaction every read it serves runs in. db has to have been
// opened read-only, or New returns ErrWritable rather than let a
// writer stall behind the transaction it is about to hold.
//
// The caller owns the result: Close ends that transaction, and db
// cannot be closed before it does.
func New(db *bolt.DB) (*Bolt, error) {
	if !db.IsReadOnly() {
		return nil, fmt.Errorf("%q: %w", db.Path(), ErrWritable)
	}
	tx, err := db.Begin(false)
	if err != nil {
		return nil, fmt.Errorf("begin read transaction: %w", err)
	}

	return &Bolt{tx: tx}, nil
}

// Close ends the transaction, and with it every listing taken from it.
// A Bolt is of no further use afterwards.
func (b *Bolt) Close() error {
	if b.tx == nil {
		return nil
	}
	tx := b.tx
	b.tx = nil
	// A read transaction is ended by rolling it back; there is
	// nothing to commit.
	if err := tx.Rollback(); err != nil {
		return fmt.Errorf("end read transaction: %w", err)
	}

	return nil
}

// Open returns a listing of the bucket at path, or of the database
// root for the empty path. A path that names a key, or names nothing,
// yields ErrNotFound: the caller asked to open something that cannot
// be opened, which is not the same as opening something empty.
func (b *Bolt) Open(path tree.Path) (tree.Listing, error) {
	cursor, err := b.cursorAt(path)
	if err != nil {
		return nil, fmt.Errorf("open %q: %w", path, err)
	}

	return &listing{cursor: cursor}, nil
}

// Read returns the value stored under path, copied out of the
// database, and the size the whole value has. At most limit bytes are
// copied; a negative limit copies all of them. A path that names a
// bucket, or names nothing, yields ErrNotFound.
func (b *Bolt) Read(path tree.Path, limit int) ([]byte, int, error) {
	bucket, err := b.bucketAt(path.Parent())
	if err != nil {
		return nil, 0, fmt.Errorf("read %q: %w", path, err)
	}
	// Get answers nil for a key that is missing and for one that is a
	// nested bucket, which a leaf never is.
	value := bucket.Get([]byte(path.Name()))
	if value == nil {
		return nil, 0, fmt.Errorf("read %q: %w", path, ErrNotFound)
	}

	size := len(value)
	if limit >= 0 && limit < size {
		value = value[:limit]
	}
	// The bytes are the memory-mapped file and are only ours until the
	// transaction ends, so the caller gets a copy of its own.
	data := make([]byte, len(value))
	copy(data, value)

	return data, size, nil
}

// cursorAt returns a cursor over the bucket at path, or over the
// database root for the empty path.
func (b *Bolt) cursorAt(path tree.Path) (*bolt.Cursor, error) {
	if len(path) == 0 {
		return b.tx.Cursor(), nil
	}
	bucket, err := b.bucketAt(path)
	if err != nil {
		return nil, err
	}

	return bucket.Cursor(), nil
}

// bucketAt walks path from the root and returns the bucket it names.
// The empty path has no bucket of its own: the database root holds
// buckets, not keys.
func (b *Bolt) bucketAt(path tree.Path) (*bolt.Bucket, error) {
	if len(path) == 0 {
		return nil, fmt.Errorf("the database root is not a bucket: %w", ErrNotFound)
	}
	bucket := b.tx.Bucket([]byte(path[0]))
	if bucket == nil {
		return nil, fmt.Errorf("bucket %q: %w", path[:1], ErrNotFound)
	}
	for i, name := range path[1:] {
		bucket = bucket.Bucket([]byte(name))
		if bucket == nil {
			return nil, fmt.Errorf("bucket %q: %w", path[:i+2], ErrNotFound)
		}
	}

	return bucket, nil
}

// listing walks one bucket with a bbolt cursor.
//
// The cursor is the position, with one thing kept alongside it: the
// name of the entry it sits on. A bbolt cursor that walks off an end
// stays there, while a tree.Listing is asked to stay where it was, so
// the name is what a move that found nothing seeks back to. Keys are
// unique and the snapshot does not change, so seeking back always
// lands on the entry it left.
type listing struct {
	cursor *bolt.Cursor
	at     string
	placed bool
	closed bool
}

func (l *listing) First() (tree.Entry, bool) {
	if l.closed {
		return tree.Entry{}, false
	}

	return l.place(l.cursor.First())
}

func (l *listing) Last() (tree.Entry, bool) {
	if l.closed {
		return tree.Entry{}, false
	}

	return l.place(l.cursor.Last())
}

// Seek places the listing on the first key that is not less than name.
// bbolt compares keys as bytes, which is the order a listing walks, so
// seeking a name that is there lands on it.
func (l *listing) Seek(name string) (tree.Entry, bool) {
	if l.closed {
		return tree.Entry{}, false
	}

	return l.place(l.cursor.Seek([]byte(name)))
}

func (l *listing) Next() (tree.Entry, bool) {
	if l.closed || !l.placed {
		return tree.Entry{}, false
	}

	return l.place(l.cursor.Next())
}

func (l *listing) Prev() (tree.Entry, bool) {
	if l.closed || !l.placed {
		return tree.Entry{}, false
	}

	return l.place(l.cursor.Prev())
}

// Close drops the cursor. There is nothing to release: the cursor
// borrows the transaction, and the transaction is the Bolt's to end.
func (l *listing) Close() error {
	l.closed = true
	l.cursor = nil

	return nil
}

// place records where a move landed and describes what it found. A
// move that found nothing puts the cursor back on the entry the
// listing was on, so running into an end does not lose the position.
func (l *listing) place(key, value []byte) (tree.Entry, bool) {
	if key == nil {
		l.rewind()

		return tree.Entry{}, false
	}
	// The key is the memory-mapped file and is only ours until the
	// transaction ends; converting it to a string copies it.
	l.at, l.placed = string(key), true

	return tree.Entry{Name: l.at, Kind: kindOf(value)}, true
}

// rewind puts the cursor back on the entry the listing is on, after a
// move that found nothing. A listing that was never placed has nowhere
// to go back to, and is left as it was.
func (l *listing) rewind() {
	if !l.placed {
		return
	}
	l.cursor.Seek([]byte(l.at))
}

// kindOf reads what a cursor said about an entry. bbolt signals a
// nested bucket with a nil value, and gives an empty value as an empty
// slice rather than nil, so the two never blur.
func kindOf(value []byte) tree.Kind {
	if value == nil {
		return tree.Node
	}

	return tree.Leaf
}

var (
	_ tree.Tree    = (*Bolt)(nil)
	_ tree.Listing = (*listing)(nil)
)
