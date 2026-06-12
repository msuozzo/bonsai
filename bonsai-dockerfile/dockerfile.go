package bonsaidockerfile

import "github.com/msuozzo/bonsai"

// Type aliases so callers can import only this package.
type (
	Parser = bonsai.Parser
	Node   = bonsai.Node
	Point  = bonsai.Point
)

// NewParser instantiates a tree-sitter Dockerfile parser.
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
	return bonsai.NewFromModule(mod, mem, mod.Xtree_sitter_dockerfile())
}
