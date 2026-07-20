// Package bonsaitsx provides a tree-sitter TSX parser built on the
// bonsai snapshot runtime. For plain .ts sources, use the sibling
// bonsai-typescript package: the two dialects are separate grammars
// upstream.
//
// Typical use:
//
//	p := bonsaitsx.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("jsx_element") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-tsx` regenerates
// module_gen.{go,dat}, libc_gen.go, and meta_gen.go in this directory.
//
// Grammar source: https://github.com/tree-sitter/tree-sitter-typescript
package bonsaitsx
