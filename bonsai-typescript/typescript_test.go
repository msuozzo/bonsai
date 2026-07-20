package bonsaitypescript_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaitypescript "github.com/msuozzo/bonsai/bonsai-typescript"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaitypescript.Node) string {
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
	p := bonsaitypescript.NewParser()

	src := []byte("const x: number = 1\n")
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

func TestInterfaceAndFunction(t *testing.T) {
	p := bonsaitypescript.NewParser()

	src := []byte("interface Point { x: number }\nfunction add(x: number, y: number): number { return x + y }\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var iface *bonsaitypescript.Node
	for n := range root.Find("interface_declaration") {
		iface = n
		break
	}
	if iface == nil {
		t.Fatalf("no interface_declaration in: %s", sexp(root))
	}
	if got := string(iface.ChildByField("name").Text(src)); got != "Point" {
		t.Errorf("interface name = %q, want %q", got, "Point")
	}

	var fn *bonsaitypescript.Node
	for n := range root.Find("function_declaration") {
		fn = n
		break
	}
	if fn == nil {
		t.Fatalf("no function_declaration in: %s", sexp(root))
	}
	if got := string(fn.ChildByField("name").Text(src)); got != "add" {
		t.Errorf("function name = %q, want %q", got, "add")
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
	p := bonsaitypescript.NewParser()

	files := []string{
		"const x: number = 1\n",
		"type Pair<T> = [T, T]\n",
		"enum Color { Red, Green }\nexport const c: Color = Color.Red\n",
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
