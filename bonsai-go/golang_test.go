package bonsaigo_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaigo "github.com/msuozzo/bonsai/bonsai-go"
)

// sexp renders the snapshot in tree-sitter's S-expression form (named
// nodes only). Used for diagnostics in test failures.
func sexp(n *bonsaigo.Node) string {
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
	p := bonsaigo.NewParser()

	src := []byte("package main\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}
	if root.Type != "source_file" {
		t.Errorf("root type = %q, want %q", root.Type, "source_file")
	}
	if got, want := uint32(len(src)), root.EndByte; got != want {
		t.Errorf("root end byte = %d, want %d", root.EndByte, got)
	}
	t.Logf("snapshot: %s", sexp(root))
}

func TestFunctionDecl(t *testing.T) {
	p := bonsaigo.NewParser()

	src := []byte(`package main

func Add(a, b int) int {
	return a + b
}
`)
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	var fn *bonsaigo.Node
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
	if got := string(name.Text(src)); got != "Add" {
		t.Errorf("name = %q, want %q", got, "Add")
	}
}

func TestImports(t *testing.T) {
	p := bonsaigo.NewParser()

	src := []byte(`package main

import (
	"fmt"
	"strings"
)

func main() { fmt.Println(strings.ToUpper("hi")) }
`)
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	var paths []string
	for n := range root.Find("interpreted_string_literal") {
		paths = append(paths, string(n.Text(src)))
	}
	// At minimum, the two import paths plus the "hi" arg literal.
	wantContains := []string{`"fmt"`, `"strings"`, `"hi"`}
	for _, w := range wantContains {
		found := false
		for _, p := range paths {
			if p == w {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected string literal %s in %v", w, paths)
		}
	}
}

func TestReuseAcrossFiles(t *testing.T) {
	p := bonsaigo.NewParser()

	files := []string{
		"package main\n",
		"package x\n\nvar y = 1\n",
		"package main\n\nfunc main() { for i := 0; i < 3; i++ { _ = i } }\n",
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
