// Package tree is the vocabulary the rest of the program speaks: a
// database seen as a tree of named entries, each of them a node to
// open or a leaf to read.
//
// The package holds no implementation and imports nothing of the
// program. A source of data fills the tree — a bbolt database today, a
// directory tomorrow — and a user interface walks it, neither naming
// the other. Both sides agree here.
//
// # Addressing
//
// A Path names an entry by naming every node from the root down to it,
// and it is the only durable way to address one. A source is free to
// hand out handles of its own — a bbolt bucket, an open directory —
// but those die with the read they were taken from, and a path does
// not. It is also the only way to walk back up: an entry knows its
// name, never what holds it.
//
// # Walking
//
// A Listing walks what one node holds. It is a cursor rather than a
// slice, because a node here is a bucket that may hold millions of
// keys next to a pane that shows forty lines: what a walker sees, it
// pays for, and nothing else. Moving is expected to be cheap, and
// Seek is expected to be no dearer than a lookup, so a listing can be
// scrolled and searched rather than loaded.
//
// # The snapshot a tree stands on
//
// A tree is a read-only view of one state of its source, and it is
// expected to hold that state still for as long as it is open: the
// same path listed twice yields the same entries in the same order.
// This is what lets a listing stay a cursor instead of a copy. A
// source that cannot promise it has to close and reopen the tree to
// move to a newer state, because nothing here detects the difference
// and a walker will not look for one.
package tree

import "strings"

// Kind tells entries apart. It is a field rather than a type, so that
// reading an entry is a comparison and never a type assertion: a
// listing knows what it found, and says so.
type Kind int

const (
	// Unknown is the zero value: an entry a source could describe but
	// not classify. A walker shows it and refuses to act on it.
	Unknown Kind = iota
	// Node is an entry that holds entries of its own, and can be
	// opened: a bucket, a directory.
	Node
	// Leaf is an entry that holds bytes, and can be read: a key/value
	// pair, a file. What the bytes mean is nothing the tree knows.
	Leaf
)

func (k Kind) String() string {
	switch k {
	case Node:
		return "node"
	case Leaf:
		return "leaf"
	default:
		return "unknown"
	}
}

// An Entry is one element of a node. It is data and not a handle: it
// says what a listing found, and holds nothing that could go stale or
// have to be closed.
type Entry struct {
	// Name addresses the entry inside the node that holds it, and is
	// unique among that node's entries. It is a copy the holder may
	// keep, and it is raw bytes: a key of a bbolt database is any
	// byte string at all, so Name is not promised to be valid UTF-8,
	// printable, or free of the path separator. Whoever shows it is
	// the one who has to make it safe to show.
	Name string
	Kind Kind
}

// A Path names an entry by naming every node from the root down to it.
// The root itself has the empty path. The names are raw, exactly as
// Entry.Name holds them.
type Path []string

// Child names the entry called name inside the node p names. The
// result is a slice of its own, so two children of one node never
// share memory and neither can be surprised by the other.
func (p Path) Child(name string) Path {
	child := make(Path, len(p), len(p)+1)
	copy(child, p)

	return append(child, name)
}

// Parent names the node holding the entry p names. The root is its own
// parent, so walking up never runs out of tree.
func (p Path) Parent() Path {
	if len(p) == 0 {
		return p
	}

	return p[:len(p)-1]
}

// Name is the name of the entry p addresses, and is empty at the root,
// which has no name of its own.
func (p Path) Name() string {
	if len(p) == 0 {
		return ""
	}

	return p[len(p)-1]
}

// String names the path the way a person reads it, for error messages
// and logs. Names are arbitrary bytes and may hold the separator
// themselves, so the result says where something is but cannot be read
// back into a path.
func (p Path) String() string {
	if len(p) == 0 {
		return "/"
	}

	return strings.Join(p, "/")
}

// A Listing walks the entries of one node, in the order the source
// means them to be shown.
//
// It holds a position rather than a copy of the entries. Every move
// reports the entry it landed on, and whether there was one to land
// on: a listing of an empty node answers false to everything.
//
// A listing starts before the first entry, so it has to be placed by
// First, Last or Seek before Next and Prev mean anything; those two
// answer false until it is. A move that runs off an end answers false
// and leaves the position where it was, so a walker that scrolls into
// the last entry can scroll back out of it.
//
// A listing reads from the source and holds what that read needs, so
// it has to be closed. Closing it twice is allowed and does nothing.
type Listing interface {
	// First places the listing on the first entry of the node.
	First() (Entry, bool)
	// Last places the listing on the last entry of the node.
	Last() (Entry, bool)
	// Next moves one entry towards the end.
	Next() (Entry, bool)
	// Prev moves one entry towards the start.
	Prev() (Entry, bool)
	// Seek places the listing on the first entry whose name is not
	// less than name, which is the entry itself when there is one by
	// that name. It answers false when every name is less, i.e. when
	// the seek ran past the end.
	Seek(name string) (Entry, bool)
	// Close releases the listing. Moves after it are not allowed.
	Close() error
}

// A Tree is a source of entries, read at one state of itself.
type Tree interface {
	// Open walks to the node at path and returns a listing of what it
	// holds. The empty path is the root. It reads from the source, so
	// it can fail: a path that names nothing, or names a leaf, is an
	// error and not an empty listing.
	Open(path Path) (Listing, error)
	// Read returns the bytes of the leaf at path, copied out of the
	// source, so the caller may keep them, together with the size the
	// whole value has. A path that names nothing, or names a node, is
	// an error.
	//
	// A leaf here is a database value and may be larger than anything
	// a reader means to look at, so limit bounds what is read: at
	// most limit bytes come back, and a negative limit asks for the
	// whole value. A reader that asked for less than the size it was
	// told is holding a prefix, and is the one to say so.
	Read(path Path, limit int) (data []byte, size int, err error)
}
