package bonsairust_test

import (
	"fmt"
	"strings"
	"testing"

	bonsairust "github.com/msuozzo/bonsai/bonsai-rust"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsairust.Node) string {
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
	p := bonsairust.NewParser()

	src := []byte("let x = 1;\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.Type != "source_file" {
		t.Fatalf("root type = %q, want %q", root.Type, "source_file")
	}
	if got, want := uint32(len(src)), root.EndByte; got != want {
		t.Fatalf("root end byte = %d, want %d", root.EndByte, got)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	t.Logf("snapshot: %s", sexp(root))
}

func TestFunctionItem(t *testing.T) {
	p := bonsairust.NewParser()

	src := []byte("fn add(x: i32, y: i32) -> i32 {\n    x + y\n}\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var fn *bonsairust.Node
	for n := range root.Find("function_item") {
		fn = n
		break
	}
	if fn == nil {
		t.Fatalf("no function_item in: %s", sexp(root))
	}

	name := fn.ChildByField("name")
	if name == nil {
		t.Fatalf("function_item has no name field: %s", sexp(fn))
	}
	if got := string(name.Text(src)); got != "add" {
		t.Errorf("name = %q, want %q", got, "add")
	}

	params := fn.ChildByField("parameters")
	if params == nil {
		t.Fatalf("function_item has no parameters field: %s", sexp(fn))
	}
	var names []string
	for n := range params.Find("identifier") {
		names = append(names, string(n.Text(src)))
	}
	if got, want := strings.Join(names, ","), "x,y"; got != want {
		t.Errorf("parameter idents = %q, want %q", got, want)
	}
}

func TestReuseAcrossFiles(t *testing.T) {
	p := bonsairust.NewParser()

	files := []string{
		"let x = 1;\n",
		"struct Point { x: f64, y: f64 }\n",
		"impl Point { fn origin() -> Point { Point { x: 0.0, y: 0.0 } } }\n",
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
