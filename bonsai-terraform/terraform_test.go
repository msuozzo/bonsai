package bonsaiterraform_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaiterraform "github.com/msuozzo/bonsai/bonsai-terraform"
)

// sexp renders the snapshot in tree-sitter's S-expression form (named
// nodes only). Used for diagnostics in test failures.
func sexp(n *bonsaiterraform.Node) string {
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
	p := bonsaiterraform.NewParser()

	src := []byte("a = 1\n")
	root, err := p.Parse(src)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}
	if root.Type != "config_file" {
		t.Errorf("root type = %q, want %q", root.Type, "config_file")
	}
	t.Logf("snapshot: %s", sexp(root))
}

func TestResourceBlock(t *testing.T) {
	p := bonsaiterraform.NewParser()

	src := []byte(`resource "aws_s3_bucket" "b" {
  bucket = "my-bucket"
  tags = {
    env = "prod"
  }
}
`)
	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}
	if root.HasError() {
		t.Fatalf("unexpected parse error: %s", sexp(root))
	}

	var block *bonsaiterraform.Node
	for n := range root.Find("block") {
		block = n
		break
	}
	if block == nil {
		t.Fatalf("no block in: %s", sexp(root))
	}

	var idents []string
	for n := range root.Find("identifier") {
		idents = append(idents, string(n.Text(src)))
	}
	for _, want := range []string{"resource", "bucket"} {
		found := false
		for _, id := range idents {
			if id == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("expected identifier %q in %v", want, idents)
		}
	}

	var attrs int
	for range root.Find("attribute") {
		attrs++
	}
	if attrs == 0 {
		t.Errorf("no attribute nodes in: %s", sexp(root))
	}
}

func TestReuseAcrossFiles(t *testing.T) {
	p := bonsaiterraform.NewParser()

	files := []string{
		"a = 1\n",
		"variable \"region\" {\n  default = \"us-east-1\"\n}\n",
		"module \"m\" {\n  source = \"./mod\"\n}\n",
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
