// Package bonsaimarkdown provides a tree-sitter Markdown parser
// (block-level grammar) built on the bonsai snapshot runtime.
//
// Typical use:
//
//	p := bonsaimarkdown.NewParser()
//	root, err := p.Parse(src)
//	if err != nil { ... }
//	for n := range root.Find("atx_heading") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-markdown` regenerates
// module_gen.{go,dat}, libc_gen.go, and meta_gen.go in this
// directory.
//
// This package builds only the block-level grammar from
// tree-sitter-grammars/tree-sitter-markdown. The companion
// tree-sitter-markdown-inline grammar (used for parsing the text
// inside paragraphs, headings, and list items) is a separate language
// and would need its own bonsai module.
//
// Grammar source: https://github.com/tree-sitter-grammars/tree-sitter-markdown
package bonsaimarkdown
