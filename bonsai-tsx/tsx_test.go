package bonsaitsx_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaitsx "github.com/msuozzo/bonsai/bonsai-tsx"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsaitsx.Node) string {
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
	p := bonsaitsx.NewParser()

	src := []byte("const x: number = 1\n")
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

func TestJSXElement(t *testing.T) {
	p := bonsaitsx.NewParser()

	src := []byte("const el = <div className=\"a\">hi {name}</div>\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	var el *bonsaitsx.Node
	for n := range root.Find("jsx_element") {
		el = n
		break
	}
	if el == nil {
		t.Fatalf("no jsx_element in: %s", sexp(root))
	}
	t.Logf("jsx: %s", sexp(el))
}

func TestReuseAcrossFiles(t *testing.T) {
	p := bonsaitsx.NewParser()

	files := []string{
		"const x: number = 1\n",
		"const App = () => <main><h1>hello</h1></main>\n",
		"function Frame(props: { title: string }) { return <div>{props.title}</div> }\n",
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
