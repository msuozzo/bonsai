package bonsaiyaml_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaiyaml "github.com/msuozzo/bonsai/bonsai-yaml"
)

// sexp renders the snapshot in tree-sitter's S-expression form (named
// nodes only). Used for diagnostics in test failures.
func sexp(n *bonsaiyaml.Node) string {
	if !n.Named {
		return ""
	}
	var b strings.Builder
	if n.Field != "" {
		fmt.Fprintf(&b, "%s: ", n.Field)
	}
	fmt.Fprintf(&b, "(%s", n.Type)
	for _, c := range n.Children {
		s := sexp(c)
		if s != "" {
			b.WriteByte(' ')
			b.WriteString(s)
		}
	}
	b.WriteByte(')')
	return b.String()
}

func TestSmoke(t *testing.T) {
	p := bonsaiyaml.NewParser()

	src := []byte("key: value\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}
	if root.Type != "stream" {
		t.Errorf("root type = %q, want %q", root.Type, "stream")
	}
	t.Logf("snapshot: %s", sexp(root))
}

func TestMapping(t *testing.T) {
	p := bonsaiyaml.NewParser()

	src := []byte("name: bonsai\nlangs:\n  - yaml\n  - go\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	var keys []string
	for pair := range root.Find("block_mapping_pair") {
		if k := pair.ChildByField("key"); k != nil {
			keys = append(keys, string(k.Text(src)))
		}
	}
	for _, want := range []string{"name", "langs"} {
		found := false
		for _, k := range keys {
			if k == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected mapping key %q in %v", want, keys)
		}
	}

	var items int
	for range root.Find("block_sequence_item") {
		items++
	}
	if items != 2 {
		t.Errorf("block_sequence_item count = %d, want 2: %s", items, sexp(root))
	}
}

func TestReuseAcrossFiles(t *testing.T) {
	p := bonsaiyaml.NewParser()

	files := []string{
		"a: 1\n",
		"- x\n- y\n",
		"outer:\n  inner: [1, 2, 3]\n",
	}
	for _, src := range files {
		root, err := p.Parse([]byte(src))
		if err != nil {
			t.Fatalf("Parse(%q): %v", src, err)
		}
		if root.HasError() {
			t.Errorf("HasError on %q: %s", src, sexp(root))
		}
	}
}
