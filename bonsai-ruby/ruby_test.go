package bonsairuby_test

import (
	"fmt"
	"strings"
	"testing"

	bonsairuby "github.com/msuozzo/bonsai/bonsai-ruby"
)

// sexp serializes a snapshot in tree-sitter's S-expression form so we can
// diff against the upstream CLI's `tree-sitter parse` output.
//
// Named nodes appear as `(type ...)`. Unnamed (anonymous) nodes are
// omitted to match the CLI default. Field names are prefixed
// `field:` per the CLI convention.
func sexp(n *bonsairuby.Node) string {
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
	p := bonsairuby.NewParser()

	src := []byte("x = 1\n")
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

func TestMethod(t *testing.T) {
	p := bonsairuby.NewParser()

	src := []byte("def add(x, y)\n  x + y\nend\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	var m *bonsairuby.Node
	for n := range root.Find("method") {
		m = n
		break
	}
	if m == nil {
		t.Fatalf("no method in: %s", sexp(root))
	}

	name := m.ChildByField("name")
	if name == nil {
		t.Fatalf("method has no name field: %s", sexp(m))
	}
	if got := string(name.Text(src)); got != "add" {
		t.Errorf("name = %q, want %q", got, "add")
	}

	params := m.ChildByField("parameters")
	if params == nil {
		t.Fatalf("method has no parameters field: %s", sexp(m))
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
	p := bonsairuby.NewParser()

	files := []string{
		"x = 1\n",
		"class Greeter\n  def hi\n    puts 'hi'\n  end\nend\n",
		"[1, 2, 3].each { |n| puts n }\n",
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
