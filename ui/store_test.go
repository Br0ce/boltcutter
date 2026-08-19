package ui

import (
	"slices"
	"testing"

	bolt "go.etcd.io/bbolt"

	"github.com/Br0ce/boltcutter/internal/testutil"
)

// newTestStore returns a Store over a throwaway database holding a flat
// bucket and a bucket with a nested sub-bucket.
func newTestStore(t *testing.T) Store {
	t.Helper()

	db := testutil.TempDB(t)
	err := db.Update(func(tx *bolt.Tx) error {
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

		return flags.Put([]byte("beta"), []byte("false"))
	})
	if err != nil {
		t.Fatalf("populate db: %v", err)
	}

	return NewStore(db)
}

func TestEntries(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		path []string
		want []Entry
	}{
		{
			name: "root holds buckets only",
			path: nil,
			want: []Entry{{Name: "config", Bucket: true}, {Name: "users", Bucket: true}},
		},
		{
			// Nested buckets and keys are listed together, in key order.
			name: "buckets and keys mixed",
			path: []string{"config"},
			want: []Entry{{Name: "flags", Bucket: true}, {Name: "version"}},
		},
		{name: "flat bucket", path: []string{"users"}, want: []Entry{{Name: "user:001"}}},
		{name: "sub-bucket", path: []string{"config", "flags"}, want: []Entry{{Name: "beta"}}},
		{name: "missing", path: []string{"nope"}, want: nil},
		{name: "missing nested", path: []string{"config", "nope"}, want: nil},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store := newTestStore(t)
			got, err := store.Entries(test.path)
			if err != nil {
				t.Fatalf("Entries(%v): %v", test.path, err)
			}
			if !slices.Equal(got, test.want) {
				t.Errorf("Entries(%v) = %v, want %v", test.path, got, test.want)
			}
		})
	}
}

func TestValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    []string
		key     string
		want    string
		wantErr bool
	}{
		{name: "flat bucket", path: []string{"users"}, key: "user:001", want: `{"id":1}`},
		{name: "sub-bucket", path: []string{"config", "flags"}, key: "beta", want: "false"},
		{name: "unknown key", path: []string{"users"}, key: "nope", wantErr: true},
		{name: "unknown bucket", path: []string{"nope"}, key: "user:001", wantErr: true},
		{name: "root has no keys", path: nil, key: "users", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			store := newTestStore(t)
			got, err := store.Value(test.path, test.key)
			if test.wantErr {
				if err == nil {
					t.Fatalf("Value(%v, %q) = %q, want error", test.path, test.key, got)
				}

				return
			}
			if err != nil {
				t.Fatalf("Value(%v, %q): %v", test.path, test.key, err)
			}
			if string(got) != test.want {
				t.Errorf("Value(%v, %q) = %q, want %q", test.path, test.key, got, test.want)
			}
		})
	}
}
