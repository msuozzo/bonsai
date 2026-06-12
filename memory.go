package bonsai

// Memory backs the WASM module's linear memory.
//
// The portable implementation grows by appending Go-side, which can relocate
// the backing array. Every helper in marshal.go re-fetches the slice through
// Slice() before each access. Do not cache the byte slice across allocations.
type Memory struct {
	Buf []byte
	Max int64 // ceiling in 64 KiB pages. Grow returns -1 past this.
}

func (m *Memory) Slice() *[]byte { return &m.Buf }

func (m *Memory) Grow(delta, _ int64) int64 {
	n := len(m.Buf)
	old := int64(n >> 16)
	if delta == 0 {
		return old
	}
	nw := old + delta
	add := int(nw)<<16 - n
	if nw > m.Max || add < 0 {
		return -1
	}
	m.Buf = append(m.Buf, make([]byte, add)...)
	return old
}
