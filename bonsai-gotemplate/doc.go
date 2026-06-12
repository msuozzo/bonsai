// Package bonsaigotemplate provides a tree-sitter parser for Go
// text/template and html/template syntax, built on the bonsai
// snapshot runtime.
//
// Typical use:
//
//	p := bonsaigotemplate.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("action") { ... }
//
// The parser handles {{ ... }} actions, pipelines, range/if/with
// blocks, defined templates, and template inclusions. The text
// outside of action delimiters is left as opaque content nodes, so
// when embedding templates in HTML (or another host language) the
// usual move is to combine this parser with a host parser via
// Parser.With.
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-gotemplate` regenerates
// module_gen.{go,dat}, libc_gen.go, and meta_gen.go in this
// directory.
//
// Grammar source: https://github.com/ngalaiko/tree-sitter-go-template
package bonsaigotemplate
