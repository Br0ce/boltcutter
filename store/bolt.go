// Package store implements the terminal user interface for browsing a bbolt
// database. It shows the database as columns that shift left as you dive
// into nested buckets, with the rightmost column previewing whatever is
// selected.
package store

import (
	"fmt"
	"path"

	bolt "go.etcd.io/bbolt"
)

// An Entry is one element of a bucket: either a nested bucket or a
// key/value pair.
type Entry struct {
	Name   string
	Bucket bool
}

// Bolt reads a bbolt database. Every call runs in its own read-only
// transaction and copies out what it returns, because bbolt only keeps
// the memory-mapped bytes valid for the life of the transaction.
type Bolt struct {
	db *bolt.DB
}

// New returns a Store reading from db.
func New(db *bolt.DB) *Bolt {
	return &Bolt{db: db}
}

func (b *Bolt) Entries(path []string) ([]Entry, error) {
	var entries []Entry
	err := b.db.View(func(tx *bolt.Tx) error {
		cursor, err := cursorAt(tx, path)
		if err != nil {
			return err
		}
		if cursor == nil {
			return nil
		}
		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			// bbolt signals a nested bucket with a nil value.
			entries = append(entries, Entry{Name: string(k), Bucket: v == nil})
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return entries, nil
}

func (b *Bolt) Value(paths []string, key string) ([]byte, error) {
	var value []byte
	err := b.db.View(func(tx *bolt.Tx) error {
		bucket, err := bucketAt(tx, paths)
		if err != nil {
			return err
		}
		if bucket == nil {
			return fmt.Errorf("bucket %q not found", path.Join(paths...))
		}
		v := bucket.Get([]byte(key))
		if v == nil {
			return fmt.Errorf("key %q not found in bucket %q", key, path.Join(paths...))
		}
		value = make([]byte, len(v))
		copy(value, v)

		return nil
	})
	if err != nil {
		return nil, err
	}

	return value, nil
}

// cursorAt returns a cursor over the bucket at path, or over the database
// root for the empty path. It returns a nil cursor if the bucket is gone.
func cursorAt(tx *bolt.Tx, path []string) (*bolt.Cursor, error) {
	if len(path) == 0 {
		return tx.Cursor(), nil
	}
	bucket, err := bucketAt(tx, path)
	if err != nil {
		return nil, err
	}
	if bucket == nil {
		return nil, nil
	}

	return bucket.Cursor(), nil
}

// bucketAt walks path from the root and returns the bucket it names, or
// nil if any element along the way is missing. The empty path has no
// bucket of its own and yields nil.
func bucketAt(tx *bolt.Tx, path []string) (*bolt.Bucket, error) {
	if len(path) == 0 {
		return nil, nil
	}
	bucket := tx.Bucket([]byte(path[0]))
	if bucket == nil {
		return nil, nil
	}
	for _, name := range path[1:] {
		bucket = bucket.Bucket([]byte(name))
		if bucket == nil {
			return nil, nil
		}
	}

	return bucket, nil
}
