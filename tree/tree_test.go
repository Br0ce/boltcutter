package tree_test

import (
	"slices"
	"testing"

	"github.com/Br0ce/boltcutter/tree"
)

func TestPathChildDoesNotShareMemory(t *testing.T) {
	t.Parallel()

	// A path with room to spare in its backing array is what makes a
	// careless append hand two children the same memory.
	parent := make(tree.Path, 1, 8)
	parent[0] = "a"

	first := parent.Child("b")
	second := parent.Child("c")

	if want := (tree.Path{"a", "b"}); !slices.Equal(first, want) {
		t.Errorf("first child = %q, want %q", first, want)
	}
	if want := (tree.Path{"a", "c"}); !slices.Equal(second, want) {
		t.Errorf("second child = %q, want %q", second, want)
	}
}

func TestPathParentStopsAtTheRoot(t *testing.T) {
	t.Parallel()

	root := tree.Path{}.Parent()
	if len(root) != 0 {
		t.Errorf("parent of the root = %q, want the root", root)
	}

	p := tree.Path{"a", "b"}
	if want := (tree.Path{"a"}); !slices.Equal(p.Parent(), want) {
		t.Errorf("parent of %q = %q, want %q", p, p.Parent(), want)
	}
}

func TestPathName(t *testing.T) {
	t.Parallel()

	if name := (tree.Path{}).Name(); name != "" {
		t.Errorf("name of the root = %q, want the empty name", name)
	}
	if name := (tree.Path{"a", "b"}).Name(); name != "b" {
		t.Errorf("name = %q, want \"b\"", name)
	}
}

func TestPathString(t *testing.T) {
	t.Parallel()

	if s := (tree.Path{}).String(); s != "/" {
		t.Errorf("the root reads %q, want \"/\"", s)
	}
	if s := (tree.Path{"a", "b"}).String(); s != "a/b" {
		t.Errorf("path reads %q, want \"a/b\"", s)
	}
}

func TestKindString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		kind tree.Kind
		want string
	}{
		{tree.Node, "node"},
		{tree.Leaf, "leaf"},
		{tree.Unknown, "unknown"},
		{tree.Kind(42), "unknown"},
	}
	for _, test := range tests {
		if got := test.kind.String(); got != test.want {
			t.Errorf("Kind(%d) reads %q, want %q", test.kind, got, test.want)
		}
	}
}
