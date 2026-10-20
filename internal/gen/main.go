// Command gen (re)generates the bbolt fixture the test suite reads and
// the browser is pointed at by hand. The output is deterministic and is
// checked into the repository under testdata/.
//
// Regenerate with:
//
//	go generate ./...
//
// or directly, with a bucket big enough to make the browser work for
// its living:
//
//	go run ./internal/gen -events 200000
//
// The fixture is a committed binary produced by a specific bbolt
// version (see testdata/README.md). Review the size of the change
// before committing a larger one.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	bolt "go.etcd.io/bbolt"
)

// Shapes of the fixture that the browser is meant to be tried against.
const (
	// deepLevels is how many buckets the deep chain nests, for
	// watching the columns shift on the way down and back up.
	deepLevels = 8
	// hugeValueSize is bigger than the preview pane reads, so the
	// pane has to say that it is holding a prefix.
	hugeValueSize = 512 << 10
)

func main() {
	events := flag.Int("events", 50_000, "number of key/value pairs in the events bucket")
	out := flag.String("out", filepath.Join(repoRoot(), "testdata", "test.db"), "where to write the fixture")
	flag.Parse()

	if err := generate(*out, *events); err != nil {
		log.Fatal(err)
	}
	info, err := os.Stat(*out)
	if err != nil {
		log.Fatalf("stat fixture: %v", err)
	}
	fmt.Printf("wrote %s (%d events, %.1f KiB)\n", *out, *events, float64(info.Size())/1024)
}

// generate writes the fixture to out, replacing whatever was there.
//
// The database is built in a scratch file and then compacted into
// place, because bbolt doubles a file as it grows and a fixture built
// in one pass carries as much empty space as data. Compacting is what
// keeps a committed binary close to the size of what it holds.
func generate(out string, events int) error {
	if err := os.MkdirAll(filepath.Dir(out), 0o750); err != nil {
		return fmt.Errorf("create fixture dir: %w", err)
	}
	for _, path := range []string{out, out + ".raw"} {
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove old fixture: %w", err)
		}
	}
	defer os.Remove(out + ".raw")

	src, err := bolt.Open(out+".raw", 0o600, nil)
	if err != nil {
		return fmt.Errorf("open scratch db: %w", err)
	}
	defer src.Close()

	if err := populate(src, events); err != nil {
		return fmt.Errorf("populate db: %w", err)
	}

	dst, err := bolt.Open(out, 0o600, nil)
	if err != nil {
		return fmt.Errorf("open fixture: %w", err)
	}
	defer dst.Close()

	if err := bolt.Compact(dst, src, 0); err != nil {
		return fmt.Errorf("compact fixture: %w", err)
	}

	return nil
}

// populate fills db with the shapes the browser has to cope with: a
// bucket far longer than any pane, values that are no JSON or too long
// to show, buckets nested deeper than the three columns, and keys whose
// names a terminal cannot be shown as they are.
func populate(db *bolt.DB, events int) error {
	return db.Update(func(tx *bolt.Tx) error {
		for _, fill := range []func(*bolt.Tx) error{
			fillUsers,
			fillConfig,
			func(tx *bolt.Tx) error { return fillEvents(tx, events) },
			fillBlobs,
			fillDeep,
		} {
			if err := fill(tx); err != nil {
				return err
			}
		}

		return nil
	})
}

// fillUsers writes the flat bucket of JSON values.
func fillUsers(tx *bolt.Tx) error {
	users, err := tx.CreateBucketIfNotExists([]byte("users"))
	if err != nil {
		return err
	}
	for i := 1; i <= 5; i++ {
		value := map[string]any{
			"id":    i,
			"name":  fmt.Sprintf("John Doe %d", i),
			"email": fmt.Sprintf("user%d@example.com", i),
		}
		if err := putJSON(users, fmt.Sprintf("user:%03d", i), value); err != nil {
			return err
		}
	}

	return nil
}

// fillConfig writes the bucket that holds both keys and a sub-bucket.
func fillConfig(tx *bolt.Tx) error {
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
}

// fillEvents writes the long bucket. It is the one the browser is
// pointed at to see that a listing costs what it shows and not what it
// holds: scrolling it, and jumping to its end, has to stay instant
// however many entries there are.
func fillEvents(tx *bolt.Tx, n int) error {
	events, err := tx.CreateBucketIfNotExists([]byte("events"))
	if err != nil {
		return err
	}
	// Keys are zero-padded so they read in the order bbolt walks them,
	// which is by bytes and not by number.
	width := len(fmt.Sprint(n - 1))
	kinds := []string{"click", "scroll", "submit", "error", "view"}
	for i := range n {
		value := map[string]any{
			"seq":  i,
			"kind": kinds[i%len(kinds)],
			"at":   fmt.Sprintf("2026-01-01T%02d:%02d:%02dZ", i/3600%24, i/60%60, i%60),
		}
		if err := putJSON(events, fmt.Sprintf("event:%0*d", width, i), value); err != nil {
			return err
		}
	}

	return nil
}

// fillBlobs writes the values that are not a tidy little JSON object:
// one too long for the preview to read whole, one that is no text at
// all, and one whose key would rewrite the screen if it were shown as
// it is.
func fillBlobs(tx *bolt.Tx) error {
	blobs, err := tx.CreateBucketIfNotExists([]byte("blobs"))
	if err != nil {
		return err
	}
	if err := blobs.Put([]byte("huge.json"), hugeJSON()); err != nil {
		return err
	}
	if err := blobs.Put([]byte("plain.txt"), []byte("not json, just a line of text")); err != nil {
		return err
	}

	// Every byte value there is, which is neither text nor any format
	// we decode.
	binary := make([]byte, 256)
	for i := range binary {
		binary[i] = byte(i)
	}
	if err := blobs.Put([]byte("binary.bin"), binary); err != nil {
		return err
	}

	// A key carrying a newline and an escape sequence, which the
	// listing has to render harmless.
	return blobs.Put([]byte("weird\nname\x1b[31m\xff"), []byte(`"a key that fights back"`))
}

// hugeJSON is a JSON array longer than the preview pane reads, so the
// pane shows what it was given rather than the whole value.
func hugeJSON() []byte {
	var out strings.Builder
	out.WriteString(`{"lines":[`)
	for i := 0; out.Len() < hugeValueSize; i++ {
		if i > 0 {
			out.WriteString(",")
		}
		fmt.Fprintf(&out, `{"n":%d,"text":"line %d of a value too long to show"}`, i, i)
	}
	out.WriteString("]}")

	return []byte(out.String())
}

// fillDeep writes a chain of buckets deeper than the browser has
// columns, so the panes can be watched shifting along.
func fillDeep(tx *bolt.Tx) error {
	bucket, err := tx.CreateBucketIfNotExists([]byte("deep"))
	if err != nil {
		return err
	}
	for level := 1; level <= deepLevels; level++ {
		if err := bucket.Put([]byte("depth"), fmt.Appendf(nil, "%d", level)); err != nil {
			return err
		}
		if level == deepLevels {
			return bucket.Put([]byte("bottom"), []byte(`"nothing below this"`))
		}
		if bucket, err = bucket.CreateBucketIfNotExists(fmt.Appendf(nil, "level-%02d", level+1)); err != nil {
			return err
		}
	}

	return nil
}

// putJSON stores value as the JSON the browser is meant to render.
func putJSON(bucket *bolt.Bucket, key string, value any) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return err
	}

	return bucket.Put([]byte(key), encoded)
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
