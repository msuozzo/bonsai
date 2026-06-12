package bonsai

import "encoding/binary"

// Every helper re-fetches *p.mem.Slice() via buf() so that a WASM-side grow
// which relocated the backing array can't leave us reading stale memory.
// The cost is a pointer deref per access. Do not optimize it away.
func (p *Parser) buf() []byte { return *p.mem.Slice() }

func (p *Parser) read32(ptr int32) uint32 {
	return binary.LittleEndian.Uint32(p.buf()[ptr:])
}

// readCStr reads a NUL-terminated C string starting at ptr.
// ptr == 0 returns "" (tree-sitter returns NULL for some missing strings).
func (p *Parser) readCStr(ptr int32) string {
	if ptr == 0 {
		return ""
	}
	b := p.buf()[ptr:]
	for i := range b {
		if b[i] == 0 {
			return string(b[:i])
		}
	}
	panic("bonsai: unterminated C string")
}

func (p *Parser) writeBytes(ptr int32, b []byte) {
	copy(p.buf()[ptr:], b)
}

func (p *Parser) alloc(n int32) int32 {
	ptr := p.mod.Xmalloc(n)
	if ptr == 0 && n != 0 {
		panic("bonsai: out of memory")
	}
	return ptr
}

func (p *Parser) free(ptr int32) {
	if ptr != 0 {
		p.mod.Xfree(ptr)
	}
}

// scratch holds preallocated linear-memory slots for the by-value structs
// the C ABI returns via a caller-provided result pointer (sret).
//
//	node:   24 B (TSNode)
//	point:   8 B (TSPoint)
//	cursor: 32 B (TSTreeCursor, actual struct is ~20 B, over-allocate)
type scratch struct {
	node   int32
	point  int32
	cursor int32
}

func (p *Parser) newScratch() scratch {
	return scratch{
		node:   p.alloc(24),
		point:  p.alloc(8),
		cursor: p.alloc(32),
	}
}

func (p *Parser) freeScratch(s scratch) {
	p.free(s.node)
	p.free(s.point)
	p.free(s.cursor)
}

func (p *Parser) readPoint(ptr int32) Point {
	return Point{
		Row: p.read32(ptr),
		Col: p.read32(ptr + 4),
	}
}
