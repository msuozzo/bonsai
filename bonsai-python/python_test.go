package bonsaipython_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaipython "github.com/msuozzo/bonsai/bonsai-python"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaipython.Node) string {
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
	p := bonsaipython.NewParser()

	src := []byte("x = 1\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.Type != "module" {
		t.Fatalf("root type = %q, want %q", root.Type, "module")
	}
	if got, want := uint32(len(src)), root.EndByte; got != want {
		t.Fatalf("root end byte = %d, want %d", root.EndByte, got)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	t.Logf("snapshot: %s", sexp(root))
}

func TestAssignmentFields(t *testing.T) {
	p := bonsaipython.NewParser()

	src := []byte("greeting = 'hi'\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var assign *bonsaipython.Node
	for n := range root.Find("assignment") {
		assign = n
		break
	}
	if assign == nil {
		t.Fatalf("no assignment found in: %s", sexp(root))
	}

	left := assign.ChildByField("left")
	right := assign.ChildByField("right")
	if left == nil || right == nil {
		t.Fatalf("missing left/right field on assignment: %s", sexp(assign))
	}
	if got := string(left.Text(src)); got != "greeting" {
		t.Errorf("left.Text = %q, want %q", got, "greeting")
	}
	if got := string(right.Text(src)); got != "'hi'" {
		t.Errorf("right.Text = %q, want %q", got, "'hi'")
	}
}

func TestFunctionDef(t *testing.T) {
	p := bonsaipython.NewParser()

	src := []byte("def f(x, y):\n    return x + y\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var fn *bonsaipython.Node
	for n := range root.Find("function_definition") {
		fn = n
		break
	}
	if fn == nil {
		t.Fatalf("no function_definition in: %s", sexp(root))
	}

	name := fn.ChildByField("name")
	if name == nil {
		t.Fatalf("function_definition has no name field: %s", sexp(fn))
	}
	if got := string(name.Text(src)); got != "f" {
		t.Errorf("name = %q, want %q", got, "f")
	}

	params := fn.ChildByField("parameters")
	if params == nil {
		t.Fatalf("function_definition has no parameters field")
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
	p := bonsaipython.NewParser()

	files := []string{
		"a = 1\n",
		"def g(): pass\n",
		"import os\nfor i in range(3): print(i)\n",
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
