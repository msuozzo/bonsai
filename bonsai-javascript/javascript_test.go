package bonsaijavascript_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaijavascript "github.com/msuozzo/bonsai/bonsai-javascript"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaijavascript.Node) string {
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
	p := bonsaijavascript.NewParser()

	src := []byte("const x = 1\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.Type != "program" {
		t.Fatalf("root type = %q, want %q", root.Type, "program")
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
	p := bonsaijavascript.NewParser()

	src := []byte("function add(x, y) { return x + y }\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var fn *bonsaijavascript.Node
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

	params := fn.ChildByField("parameters")
	if params == nil {
		t.Fatalf("function_declaration has no parameters field: %s", sexp(fn))
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
	p := bonsaijavascript.NewParser()

	files := []string{
		"const x = 1\n",
		"class Greeter { hi() { return 'hi' } }\n",
		"const add = (x, y) => x + y\nexport default add\n",
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
