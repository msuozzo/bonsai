package bonsaijava_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaijava "github.com/msuozzo/bonsai/bonsai-java"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaijava.Node) string {
	if !n.Named {
		return ""
	}
	var b strings.Builder
	if n.Field != "" {
		fmt.Fprintf(&b, "%s: ", n.Field)
	}
	fmt.Fprintf(&b, "(%s", n.Kind)
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
	p := bonsaijava.NewParser()

	src := []byte("class A {}\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.Kind != "program" {
		t.Fatalf("root kind = %q, want %q", root.Kind, "program")
	}
	if got, want := uint32(len(src)), root.EndByte; got != want {
		t.Fatalf("root end byte = %d, want %d", root.EndByte, got)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	t.Logf("snapshot: %s", sexp(root))
}

func TestClassAndMethod(t *testing.T) {
	p := bonsaijava.NewParser()

	src := []byte("class Greeter {\n  int add(int x, int y) { return x + y; }\n}\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var cls *bonsaijava.Node
	for n := range root.Find("class_declaration") {
		cls = n
		break
	}
	if cls == nil {
		t.Fatalf("no class_declaration in: %s", sexp(root))
	}
	if got := string(cls.ChildByField("name").Text(src)); got != "Greeter" {
		t.Errorf("class name = %q, want %q", got, "Greeter")
	}

	var method *bonsaijava.Node
	for n := range root.Find("method_declaration") {
		method = n
		break
	}
	if method == nil {
		t.Fatalf("no method_declaration in: %s", sexp(root))
	}
	if got := string(method.ChildByField("name").Text(src)); got != "add" {
		t.Errorf("method name = %q, want %q", got, "add")
	}

	params := method.ChildByField("parameters")
	if params == nil {
		t.Fatalf("method_declaration has no parameters field: %s", sexp(method))
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
	p := bonsaijava.NewParser()

	files := []string{
		"class A {}\n",
		"interface B { void f(); }\n",
		"enum C { X, Y }\n",
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
