package bonsaimarkdown

import (
	"github.com/msuozzo/bonsai"
	bonsaimarkdowninline "github.com/msuozzo/bonsai/bonsai-markdown-inline"
)

// Type aliases so callers can import only this package.
type (
	Parser = bonsai.Parser
	Node   = bonsai.Node
	Point  = bonsai.Point
)

// NewParser instantiates a tree-sitter Markdown parser (block grammar
// only). The block grammar parses document structure (headings, lists,
// code blocks, block quotes) and leaves the text of paragraphs and
// headings inside opaque `inline` nodes. For a parser that also
// tokenizes inline content (links, code spans, emphasis), use
// NewFullParser.
//
// The parser owns one wasm module instance and is NOT safe for concurrent
// use. Pool a Parser per goroutine. Reuse across files. Instantiation is
// the expensive part.
func NewParser() *Parser {
	mem := &bonsai.Memory{Max: 1 << 20}
	// The generated New() copies the embedded data segment straight into
	// linear memory without growing first. Pre-allocate to the module's
	// declared initial size so the copy is in bounds.
	mem.Grow(InitialPages, 0)
	env := &bonsai.HostEnv{Mem: mem}
	mod := New(env)
	return bonsai.NewFromModule(mod, mem, mod.Xtree_sitter_markdown())
}

// NewFullParser instantiates a Markdown parser that combines the block
// and inline grammars behind a single Parser handle. The returned
// parser dispatches `inline` block-grammar nodes to the inline grammar
// and stitches the resulting tokens (links, code spans, emphasis)
// back into one unified tree.
func NewFullParser() *Parser {
	return NewParser().With(bonsai.SubParser{
		Match:  func(n *bonsai.Node) bool { return n.Named && n.Type == "inline" },
		Parser: bonsaimarkdowninline.NewParser(),
	})
}
