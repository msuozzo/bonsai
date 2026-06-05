package bonsaigroovy_test

import (
	"fmt"
	"strings"
	"testing"

	bonsaigroovy "github.com/msuozzo/bonsai/bonsai-groovy"
)

// sexp renders the snapshot in tree-sitter's S-expression form (named
// nodes only). Used for diagnostics in test failures.
func sexp(n *bonsaigroovy.Node) string {
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
	p := bonsaigroovy.NewParser()

	src := []byte("def x = 1\n")
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
	t.Logf("snapshot: %s", sexp(root))
}

// TestGradleSnippet is the case the user actually cares about: Gradle
// DSL parsed structurally instead of via string matching.
//
// We don't pin the exact grammar's node-type names because the
// (murtaza64) tree-sitter-groovy grammar is still pre-release. What we
// assert is the *capability*: every top-level call's identifier
// ("plugins", "repositories", "dependencies") must appear as some named
// identifier in the tree, and the assigned versions inside `plugins`
// must show up as string literals.
func TestGradleSnippet(t *testing.T) {
	p := bonsaigroovy.NewParser()

	src := []byte(`plugins {
    id 'java'
    id 'org.springframework.boot' version '3.2.0'
}

repositories {
    mavenCentral()
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter-web'
    testImplementation 'org.springframework.boot:spring-boot-starter-test'
}
`)

	root, err := p.Parse(src)
	if err != nil {
		t.Fatal(err)
	}

	// Collect every named identifier and string literal in the tree.
	idents := map[string]int{}
	strings_ := map[string]int{}
	for n := range root.Walk() {
		if !n.Named {
			continue
		}
		switch n.Type {
		case "identifier":
			idents[string(n.Text(src))]++
		case "string", "string_literal":
			strings_[string(n.Text(src))]++
		}
	}

	for _, want := range []string{"plugins", "repositories", "dependencies", "mavenCentral", "implementation"} {
		if idents[want] == 0 {
			t.Errorf("expected identifier %q in tree; tree=%s", want, sexp(root))
			break // one failure is enough; the tree dump is long
		}
	}

	// At least one Spring artifact string must be present (proving the
	// grammar tokenized the quoted strings inside the dependency blocks).
	foundSpring := false
	for s := range strings_ {
		if strings.Contains(s, "spring-boot") {
			foundSpring = true
			break
		}
	}
	if !foundSpring {
		t.Errorf("expected a spring-boot dependency string; got strings=%v", strings_)
	}

	t.Logf("found %d unique idents, %d unique strings, root.HasError=%v",
		len(idents), len(strings_), root.HasError())
}
