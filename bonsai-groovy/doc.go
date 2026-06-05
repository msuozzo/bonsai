// Package bonsaigroovy provides a tree-sitter Groovy parser built on the
// bonsai snapshot runtime. Useful for parsing Gradle build files (which
// are Groovy under the hood), historically a string-comparison job in Go.
//
// Typical use:
//
//	p := bonsaigroovy.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("function_declaration") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-groovy` regenerates module_gen.{go,dat},
// libc_gen.go, and meta_gen.go in this directory.
//
// Grammar source: https://github.com/murtaza64/tree-sitter-groovy
package bonsaigroovy
