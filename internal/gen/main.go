// Command gen (re)generates the golden bbolt fixture used by the test
// suite. The output is a small, deterministic database that is checked
// into the repository under testdata/.
//
// Regenerate with:
//
//	go generate ./...
//
// or directly:
//
//	go run ./internal/gen
//
// Keep the fixture small and review its contents deliberately: it is a
// committed binary produced by a specific bbolt version (see
// testdata/README.md).
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	bolt "go.etcd.io/bbolt"
)

func main() {
	out := filepath.Join(repoRoot(), "testdata", "test.db")

	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		log.Fatalf("create testdata dir: %v", err)
	}
	if err := os.RemoveAll(out); err != nil {
		log.Fatalf("remove old db: %v", err)
	}

	db, err := bolt.Open(out, 0o600, nil)
	if err != nil {
		log.Fatalf("open db: %v", err)
	}
	defer db.Close()

	if err := populate(db); err != nil {
		log.Fatalf("populate db: %v", err)
	}

	fmt.Printf("wrote %s\n", out)
}

// populate fills db with a small, deterministic set of test data: a flat
// bucket of key/value pairs and a bucket containing a nested sub-bucket.
func populate(db *bolt.DB) error {
	return db.Update(func(tx *bolt.Tx) error {
		users, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		for i := 1; i <= 5; i++ {
			key := fmt.Sprintf("user:%03d", i)
			val := fmt.Sprintf(`{"id":%d,"name":"John Doe %d","email":"user%d@example.com"}`, i, i, i)
			if err := users.Put([]byte(key), []byte(val)); err != nil {
				return err
			}
		}

		config, err := tx.CreateBucketIfNotExists([]byte("config"))
		if err != nil {
			return err
		}
		if err := config.Put([]byte("version"), []byte("1")); err != nil {
			return err
		}
		if err := config.Put([]byte("enabled"), []byte("true")); err != nil {
			return err
		}

		flags, err := config.CreateBucketIfNotExists([]byte("flags"))
		if err != nil {
			return err
		}
		if err := flags.Put([]byte("beta"), []byte("false")); err != nil {
			return err
		}
		return flags.Put([]byte("verbose"), []byte("true"))
	})
}

// repoRoot returns the module root, derived from this source file's
// location so the generator writes to the same place regardless of the
// working directory it is invoked from.
func repoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		log.Fatal("cannot determine source location")
	}
	// this file lives at <root>/internal/gen/main.go
	return filepath.Join(filepath.Dir(file), "..", "..")
}
