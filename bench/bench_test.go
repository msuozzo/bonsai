// Package bench compares bonsai-go (wasm2go-translated tree-sitter)
// against github.com/tree-sitter/go-tree-sitter (cgo bindings to the
// native tree-sitter library) on Go fixtures of varying size.
//
// Both parsers run the same grammar (tree-sitter-go). The bonsai side
// also materializes a full Go-struct snapshot per parse, which the cgo
// side does NOT. See BenchmarkCgoParseAndWalk for the apples-to-apples
// comparison that charges both sides for touching every node.
//
// The cgo dependency lives in this submodule rather than in bonsai's
// main go.mod so anyone importing bonsai or its grammar modules never
// inherits a cgo build requirement.
package bench

import (
	_ "embed"
	"testing"

	bonsaigo "github.com/msuozzo/bonsai/bonsai-go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

//go:embed testdata/tiny.go
var tinySrc []byte

//go:embed testdata/small.go
var smallSrc []byte

//go:embed testdata/large.go
var largeSrc []byte

var fixtures = []struct {
	name string
	src  []byte
}{
	{"tiny", tinySrc},
	{"small", smallSrc},
	{"large", largeSrc},
}

// walkBonsai forces a full preorder traversal of the snapshot so we
// charge the cost of touching every node, matching what the cgo side
// does via cursor iteration.
func walkBonsai(n *bonsaigo.Node) int {
	c := 1
	for _, ch := range n.Children {
		c += walkBonsai(ch)
	}
	return c
}

// walkCgo walks the cgo tree via a cursor (the same access pattern
// bonsai uses internally) so both sides do equivalent work post-parse.
func walkCgo(n *tree_sitter.Node) int {
	c := 1
	for i := uint(0); i < uint(n.ChildCount()); i++ {
		c += walkCgo(n.Child(i))
	}
	return c
}

// BenchmarkBonsaiParse measures Parse including the snapshot
// materialization. bonsai's design is "parse, walk, materialize Go
// structs, then free the tree". There's no zero-copy live-tree mode,
// so this is the only mode it offers.
func BenchmarkBonsaiParse(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			p := bonsaigo.NewParser()
			b.SetBytes(int64(len(f.src)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				root, err := p.Parse(f.src)
				if err != nil {
					b.Fatal(err)
				}
				if walkBonsai(root) == 0 {
					b.Fatal("empty tree")
				}
			}
		})
	}
}

// BenchmarkCgoParse measures the cgo binding's Parse without any
// materialization. It is the cheapest mode the binding offers.
func BenchmarkCgoParse(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			parser := tree_sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_go.Language()))
			b.SetBytes(int64(len(f.src)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree := parser.Parse(f.src, nil)
				if tree == nil {
					b.Fatal("nil tree")
				}
				tree.Close()
			}
		})
	}
}

// BenchmarkCgoParseAndWalk is the apples-to-apples comparison: both
// sides pay for parse + touching every node. The cgo binding still
// skips the materialize-Go-structs cost, so it is a strict lower
// bound on what an equivalent-API cgo wrapper would cost.
func BenchmarkCgoParseAndWalk(b *testing.B) {
	for _, f := range fixtures {
		b.Run(f.name, func(b *testing.B) {
			parser := tree_sitter.NewParser()
			defer parser.Close()
			parser.SetLanguage(tree_sitter.NewLanguage(tree_sitter_go.Language()))
			b.SetBytes(int64(len(f.src)))
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				tree := parser.Parse(f.src, nil)
				if tree == nil {
					b.Fatal("nil tree")
				}
				if walkCgo(tree.RootNode()) == 0 {
					b.Fatal("empty tree")
				}
				tree.Close()
			}
		})
	}
}
