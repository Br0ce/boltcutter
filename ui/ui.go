package ui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/Br0ce/boltcutter/store"
)

// Store is the read-only view of a bbolt database the UI works against.
//
// A path addresses a bucket by naming every bucket from the root down to
// it; the empty path addresses the database root, which holds buckets but
// no keys.
type Store interface {
	// Entries returns the contents of the bucket at path in key order,
	// nested buckets and key/value pairs alike.
	Entries(path []string) ([]store.Entry, error)
	// Value returns the value stored under key in the bucket at path.
	Value(path []string, key string) ([]byte, error)
}

// New returns a Model browsing store, loaded with the database root.
// dbPath names the file it reads and is shown in the header.
func New(store Store, dbPath string) Model {
	m := Model{
		store:  store,
		dbPath: dbPath,
		styles: newStyles(),
		focus:  paneCurrent,
		levels: []level{{}},
	}
	m.reload()

	return m
}

// Run starts the browser on the database at dbPath and blocks until the
// user quits.
func Run(store Store, dbPath string) error {
	_, err := tea.NewProgram(New(store, dbPath), tea.WithAltScreen()).Run()

	return err
}
