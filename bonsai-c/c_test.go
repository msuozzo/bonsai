package bonsaic_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaic "github.com/msuozzo/bonsai/bonsai-c"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaic.Node) string {
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
	p := bonsaic.NewParser()

	src := []byte("int x = 1;\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if root.Type != "translation_unit" {
		t.Fatalf("root type = %q, want %q", root.Type, "translation_unit")
	}
	if got, want := uint32(len(src)), root.EndByte; got != want {
		t.Fatalf("root end byte = %d, want %d", root.EndByte, got)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	t.Logf("snapshot: %s", sexp(root))
}

func TestFunctionDefinition(t *testing.T) {
	p := bonsaic.NewParser()

	src := []byte("int add(int x, int y) { return x + y; }\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var fn *bonsaic.Node
	for n := range root.Find("function_definition") {
		fn = n
		break
	}
	if fn == nil {
		t.Fatalf("no function_definition in: %s", sexp(root))
	}

	// The function name sits behind two declarator fields:
	// (function_definition declarator: (function_declarator
	//   declarator: (identifier) parameters: (parameter_list ...))).
	fd := fn.ChildByField("declarator")
	if fd == nil || fd.Type != "function_declarator" {
		t.Fatalf("function_definition has no function_declarator: %s", sexp(fn))
	}
	name := fd.ChildByField("declarator")
	if name == nil {
		t.Fatalf("function_declarator has no declarator field: %s", sexp(fd))
	}
	if got := string(name.Text(src)); got != "add" {
		t.Errorf("name = %q, want %q", got, "add")
	}

	params := fd.ChildByField("parameters")
	if params == nil {
		t.Fatalf("function_declarator has no parameters field: %s", sexp(fd))
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
	p := bonsaic.NewParser()

	files := []string{
		"int x = 1;\n",
		"struct point { int x; int y; };\n",
		"typedef unsigned int uint;\n#define MAX 10\n",
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
