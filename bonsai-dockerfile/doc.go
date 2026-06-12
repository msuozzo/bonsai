// Package bonsaidockerfile provides a tree-sitter Dockerfile parser
// built on the bonsai snapshot runtime.
//
// Typical use:
//
//	p := bonsaidockerfile.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("from_instruction") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-dockerfile` regenerates
// module_gen.{go,dat}, libc_gen.go, and meta_gen.go in this
// directory.
//
// Grammar source: https://github.com/camdencheek/tree-sitter-dockerfile
package bonsaidockerfile
