// Package bonsaitypescript provides a tree-sitter TypeScript parser
// built on the bonsai snapshot runtime. For .tsx sources, use the
// sibling bonsai-tsx package: the two dialects are separate grammars
// upstream.
//
// Typical use:
//
//	p := bonsaitypescript.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("interface_declaration") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-typescript` regenerates
// module_gen.{go,dat}, libc_gen.go, and meta_gen.go in this directory.
//
// Grammar source: https://github.com/tree-sitter/tree-sitter-typescript
package bonsaitypescript
