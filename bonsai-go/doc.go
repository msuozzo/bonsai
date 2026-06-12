// Package bonsaigo provides a tree-sitter Go parser built on the bonsai
// snapshot runtime.
//
// Typical use:
//
//	p := bonsaigo.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("function_declaration") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-go` regenerates module_gen.{go,dat},
// libc_gen.go, and meta_gen.go in this directory.
//
// Grammar source: https://github.com/tree-sitter/tree-sitter-go
package bonsaigo
