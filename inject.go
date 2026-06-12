package bonsai

// injectFn maps a node to the parser whose grammar should replace that
// node's subtree. Return nil to leave the node alone. Called for every
// node in the tree (including nested ones inside an already-injected
// subtree). Beware infinite recursion if your callback keeps matching
// nodes that come out of the same grammar it injected.
type injectFn func(n *Node) *Parser

// parseWithInject parses src and recursively re-parses any node where
// inject returns a non-nil Parser. The returned tree has the injected
// sub-trees stitched in at the appropriate byte and point offsets, so
// downstream code can walk one unified tree without knowing which
// grammar produced which node.
//
// Common use is Markdown, where the block grammar leaves paragraph
// and heading text inside opaque `inline` nodes and the companion
// markdown-inline grammar tokenizes those into links, code spans,
// emphasis, and so on. The lighter-weight form for that case is
// Parser.With, which stores the rule on the parser so plain Parse
// applies it automatically:
//
//	root, err := blockParser.parseWithInject(src, func(n *bonsai.Node) *bonsai.Parser {
//		if n.Type == "inline" {
//			return inlineParser
//		}
//		return nil
//	})
//
// Use parseWithInject when the choice of sub-parser is dynamic (e.g.
// dispatching on a code fence's info_string). Static rule lists are
// better expressed via With.
//
// inject == nil is equivalent to plain Parse.
func (p *Parser) parseWithInject(src []byte, inject injectFn) (*Node, error) {
	root, err := p.parsePlain(src)
	if err != nil {
		return nil, err
	}
	if inject == nil {
		return root, nil
	}
	// Root-level match: replace the root with its sub-tree wholesale.
	if sub := inject(root); sub != nil {
		newRoot, err := injectAt(sub, root, src)
		if err != nil {
			return root, nil
		}
		newRoot.Parent = nil
		processInject(newRoot, src, inject)
		return newRoot, nil
	}
	processInject(root, src, inject)
	return root, nil
}

// processInject walks n's children. When inject returns a non-nil
// parser for a child, that child's subtree is replaced in place.
// Recurses into the new subtree so multiply-nested injections (e.g.
// Markdown inline content that contains a fenced code block via
// reference) are handled.
func processInject(n *Node, src []byte, inject injectFn) {
	for i, c := range n.Children {
		if sub := inject(c); sub != nil {
			if replacement, err := injectAt(sub, c, src); err == nil {
				replacement.Parent = n
				n.Children[i] = replacement
				processInject(replacement, src, inject)
				continue
			}
		}
		processInject(c, src, inject)
	}
}

// injectAt parses target.Text(src) with sub and returns the sub-tree
// with byte and point offsets shifted to occupy target's place in the
// host coordinate system.
func injectAt(sub *Parser, target *Node, src []byte) (*Node, error) {
	subRoot, err := sub.Parse(target.Text(src))
	if err != nil {
		return nil, err
	}
	shiftSubtree(subRoot, target.StartByte, target.StartPoint)
	return subRoot, nil
}

func shiftSubtree(n *Node, byteOff uint32, point Point) {
	n.StartByte += byteOff
	n.EndByte += byteOff
	shiftPoint(&n.StartPoint, point)
	shiftPoint(&n.EndPoint, point)
	for _, c := range n.Children {
		shiftSubtree(c, byteOff, point)
	}
}

// shiftPoint applies host-start offsets to a sub-parse point. Row is
// always shifted by host.Row. Col is only shifted on the first row of
// the sub-parse, because subsequent rows start at column 0 in their
// own coordinate system and stay at column 0 in the host's.
func shiftPoint(p *Point, host Point) {
	if p.Row == 0 {
		p.Col += host.Col
	}
	p.Row += host.Row
}

// SubParser pairs a node predicate with the parser that should re-
// parse the matched node's text. The first SubParser whose Match
// returns true wins for any given node.
type SubParser struct {
	Match  func(*Node) bool
	Parser *Parser
}

// With registers sub-parsers on p. After With returns, p.Parse(src)
// applies the rules automatically and returns one unified tree.
// Returns p for chaining.
//
// Markdown is the canonical case (block grammar dispatching to inline
// grammar):
//
//	full := bonsaimarkdown.NewParser().With(bonsai.SubParser{
//		Match:  func(n *bonsai.Node) bool { return n.Type == "inline" },
//		Parser: bonsaimarkdowninline.NewParser(),
//	})
//	root, _ := full.Parse(src) // tree contains block + inline nodes
//
// Subtrees are themselves walked for further matches, so multi-layer parsing
// (e.g. HTML containing JS containing a template literal containing SQL)
// composes naturally. Beware a Match that fires on its own grammar's output:
// it would recurse forever. (Specifically, recursion only re-checks the
// CHILDREN of a replaced node, so a sub-parser whose root has the same Type as
// the original match is safe.)
func (p *Parser) With(subs ...SubParser) *Parser {
	if len(subs) == 0 {
		return p
	}
	rules := append([]SubParser(nil), subs...)
	p.inject = func(n *Node) *Parser {
		for _, s := range rules {
			if s.Match(n) {
				return s.Parser
			}
		}
		return nil
	}
	return p
}
