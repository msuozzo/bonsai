package bonsaikotlin_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaikotlin "github.com/msuozzo/bonsai/bonsai-kotlin"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaikotlin.Node) string {
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
	p := bonsaikotlin.NewParser()

	src := []byte("val x = 1\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.Kind != "source_file" {
		t.Fatalf("root kind = %q, want %q", root.Kind, "source_file")
	}
	if got, want := uint32(len(src)), root.EndByte; got != want {
		t.Fatalf("root end byte = %d, want %d", root.EndByte, got)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	t.Logf("snapshot: %s", sexp(root))
}

func TestFunctionDeclaration(t *testing.T) {
	p := bonsaikotlin.NewParser()

	src := []byte("fun add(x: Int, y: Int): Int = x + y\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var fn *bonsaikotlin.Node
	for n := range root.Find("function_declaration") {
		fn = n
		break
	}
	if fn == nil {
		t.Fatalf("no function_declaration in: %s", sexp(root))
	}

	name := fn.ChildByField("name")
	if name == nil {
		t.Fatalf("function_declaration has no name field: %s", sexp(fn))
	}
	if got := string(name.Text(src)); got != "add" {
		t.Errorf("name = %q, want %q", got, "add")
	}

	// Each parameter is (parameter (identifier) (user_type ...)): the
	// first identifier child is the parameter name.
	var params []string
	for pn := range fn.Find("parameter") {
		for _, c := range pn.Children {
			if c.Kind == "identifier" {
				params = append(params, string(c.Text(src)))
				break
			}
		}
	}
	if got, want := strings.Join(params, ","), "x,y"; got != want {
		t.Errorf("parameter names = %q, want %q", got, want)
	}
}

func TestReuseAcrossFiles(t *testing.T) {
	p := bonsaikotlin.NewParser()

	files := []string{
		"val x = 1\n",
		"fun f() = 2\n",
		"class Greeter(val name: String)\n",
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
