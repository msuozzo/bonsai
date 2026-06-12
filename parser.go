package bonsai

import "errors"

// ErrParseFailed is returned by Parse when tree-sitter rejects the input
// outright (returns a NULL tree pointer). The grammar is permissive enough
// that this is rare. Most malformed source yields a tree containing error
// nodes rather than a failed parse.
var ErrParseFailed = errors.New("bonsai: parse failed")

// Parser owns one wasm module instance plus a TSParser*. NOT safe for
// concurrent use. Pool a Parser per goroutine. Reuse a Parser across
// files. Module instantiation is the expensive part, and once warm a
// Parser holds onto its linear-memory high-water mark.
//
// Grammar-specific construction lives in each grammar's package
// (e.g. bonsai-python.NewParser).
type Parser struct {
	mod Module
	mem *Memory
	ts  int32 // TSParser*
}

// NewFromModule wires up a parser around an already-instantiated wasm
// module and the language pointer the grammar exposes. Called by each
// grammar package after constructing its module.
func NewFromModule(mod Module, mem *Memory, langPtr int32) *Parser {
	p := &Parser{mod: mod, mem: mem}
	// Reactor modules require an explicit _initialize call after
	// instantiation. It runs global ctors and the dlmalloc bootstrap.
	p.mod.X_initialize()
	p.ts = p.mod.Xts_parser_new()
	if p.mod.Xts_parser_set_language(p.ts, langPtr) == 0 {
		panic("bonsai: language/ABI mismatch with compiled core")
	}
	return p
}

// Parse builds an owned Node snapshot from src. After Parse returns, no
// wasm-side data backs the snapshot. The tree has been freed.
//
// Parse is not concurrent-safe with itself on the same Parser.
func (p *Parser) Parse(src []byte) (*Node, error) {
	srcPtr := p.alloc(int32(len(src)))
	p.writeBytes(srcPtr, src)

	tree := p.mod.Xts_parser_parse_string(p.ts, 0, srcPtr, int32(len(src)))
	p.free(srcPtr)
	if tree == 0 {
		return nil, ErrParseFailed
	}
	defer p.mod.Xts_tree_delete(tree)

	s := p.newScratch()
	defer p.freeScratch(s)

	p.mod.Xts_tree_root_node(s.node, tree)
	p.mod.Xts_tree_cursor_new(s.cursor, s.node)
	defer p.mod.Xts_tree_cursor_delete(s.cursor)

	return p.build(s), nil
}

// build materializes the cursor's current node and recurses into its
// children. It reads everything it needs from scratch BEFORE descending so
// that nested calls overwriting scratch can't corrupt a parent in progress.
func (p *Parser) build(s scratch) *Node {
	p.mod.Xts_tree_cursor_current_node(s.node, s.cursor)

	n := &Node{
		Type:      p.readCStr(p.mod.Xts_node_type(s.node)),
		Named:     p.mod.Xts_node_is_named(s.node) != 0,
		Field:     p.readCStr(p.mod.Xts_tree_cursor_current_field_name(s.cursor)),
		IsError:   p.mod.Xts_node_is_error(s.node) != 0,
		IsMissing: p.mod.Xts_node_is_missing(s.node) != 0,
		hasError:  p.mod.Xts_node_has_error(s.node) != 0,
		StartByte: uint32(p.mod.Xts_node_start_byte(s.node)),
		EndByte:   uint32(p.mod.Xts_node_end_byte(s.node)),
	}
	p.mod.Xts_node_start_point(s.point, s.node)
	n.StartPoint = p.readPoint(s.point)
	p.mod.Xts_node_end_point(s.point, s.node)
	n.EndPoint = p.readPoint(s.point)

	if p.mod.Xts_tree_cursor_goto_first_child(s.cursor) != 0 {
		for {
			c := p.build(s)
			c.Parent = n
			n.Children = append(n.Children, c)
			if p.mod.Xts_tree_cursor_goto_next_sibling(s.cursor) == 0 {
				break
			}
		}
		p.mod.Xts_tree_cursor_goto_parent(s.cursor)
	}
	return n
}
