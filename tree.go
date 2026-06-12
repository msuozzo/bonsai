package bonsai

import "iter"

// Point is a (row, column) source position. Columns are in bytes, not
// runes, matching the convention tree-sitter uses.
type Point struct {
	Row uint32
	Col uint32
}

// Node is a fully-owned snapshot of a tree-sitter node. There is no live
// dependency on WASM memory: after Parse returns, the tree has been deleted
// and the Go GC owns every Node.
type Node struct {
	Type       string
	Named      bool
	Field      string // field name in parent ("" if unnamed slot)
	IsError    bool   // this node is the ERROR sentinel
	IsMissing  bool   // this node was inserted to recover from a parse error
	hasError   bool   // any descendant (or self) is an error
	StartByte  uint32
	EndByte    uint32
	StartPoint Point
	EndPoint   Point
	Parent     *Node // nil at root
	Children   []*Node
}

// HasError reports whether the subtree rooted at n contains any error or
// missing nodes. Equivalent to tree-sitter's ts_node_has_error.
func (n *Node) HasError() bool { return n.hasError }

// Text returns this node's slice of the original source. The snapshot only
// stores byte offsets, and the caller keeps the source buffer.
func (n *Node) Text(src []byte) []byte { return src[n.StartByte:n.EndByte] }

// ChildByField returns the first child whose Field name equals name, or nil.
func (n *Node) ChildByField(name string) *Node {
	for _, c := range n.Children {
		if c.Field == name {
			return c
		}
	}
	return nil
}

// Find yields, in preorder, every descendant of n whose Type == typ.
// n itself is included if it matches.
func (n *Node) Find(typ string) iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		var walk func(*Node) bool
		walk = func(c *Node) bool {
			if c.Type == typ && !yield(c) {
				return false
			}
			for _, ch := range c.Children {
				if !walk(ch) {
					return false
				}
			}
			return true
		}
		walk(n)
	}
}

// Walk yields every descendant of n in preorder, including n.
func (n *Node) Walk() iter.Seq[*Node] {
	return func(yield func(*Node) bool) {
		var walk func(*Node) bool
		walk = func(c *Node) bool {
			if !yield(c) {
				return false
			}
			for _, ch := range c.Children {
				if !walk(ch) {
					return false
				}
			}
			return true
		}
		walk(n)
	}
}
