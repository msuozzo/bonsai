// Package bonsaimarkdowninline provides a tree-sitter parser for the
// inline portion of Markdown, built on the bonsai snapshot runtime.
//
// The companion block grammar (bonsai-markdown) parses document
// structure and emits opaque `inline` leaves for the text inside
// paragraphs, headings, and list items. Feeding such a leaf's text
// to this parser yields tokens for emphasis, strong emphasis, code
// spans, links, images, and the rest of the inline vocabulary.
//
// Typical use:
//
//	p := bonsaimarkdowninline.NewParser()
//	root, err := p.Parse([]byte("A *paragraph* with `code` and a [link](http://x)."))
//	if err != nil { ... }
//	for n := range root.Find("link") { ... }
//
// The Parser, Node, and Point types are aliases for the corresponding
// bonsai runtime types, so values flow naturally between bonsai-aware
// code and this package.
//
// The wasm module + grammar are generated. See Dockerfile.builder at
// the repo root. `go generate ./bonsai-markdown-inline` regenerates
// module_gen.{go,dat}, libc_gen.go, and meta_gen.go in this directory.
//
// Grammar source: https://github.com/tree-sitter-grammars/tree-sitter-markdown
// (tree-sitter-markdown-inline/ subdirectory)
package bonsaimarkdowninline
