// Package testutil provides helpers for tests that work with bbolt
// databases.
package testutil

//go:generate go run ../gen

import (
	"path/filepath"
	"testing"

	bolt "go.etcd.io/bbolt"
)

// TempDB opens a fresh, empty bbolt database in a per-test temporary
// directory and registers cleanup to close it. Use it for tests that
// need a throwaway database or that mutate their data, so they never
// touch the committed golden fixture.
func TempDB(t testing.TB) *bolt.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close temp db: %v", err)
		}
	})

	return db
}

// ReadOnlyDB builds a throwaway database with populate and reopens it
// read-only, which is how the program opens a database it browses: a
// store holds a read transaction open, and a writer sharing the file
// with one stalls behind it. Cleanup closes the database.
func ReadOnlyDB(t testing.TB, populate func(tx *bolt.Tx) error) *bolt.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "test.db")
	db, err := bolt.Open(path, 0o600, nil)
	if err != nil {
		t.Fatalf("open temp db: %v", err)
	}
	if err := db.Update(populate); err != nil {
		t.Fatalf("populate temp db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close temp db: %v", err)
	}

	readOnly, err := bolt.Open(path, 0o600, &bolt.Options{ReadOnly: true})
	if err != nil {
		t.Fatalf("reopen temp db read-only: %v", err)
	}
	t.Cleanup(func() {
		if err := readOnly.Close(); err != nil {
			t.Errorf("close temp db: %v", err)
		}
	})

	return readOnly
}
