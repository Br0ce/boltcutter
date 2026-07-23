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
