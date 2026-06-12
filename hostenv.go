package bonsai

// MemoryAPI is the interface the wasm-module side calls when it needs
// linear memory. Declared as a type alias (note the `=`) so that, when
// HostEnv.Xmemory returns MemoryAPI, the return type unifies with each
// grammar's generated `Memory` alias and HostEnv structurally satisfies
// every grammar's Xenv interface.
type MemoryAPI = interface {
	Slice() *[]byte
	Grow(delta, max int64) int64
}

// HostEnv satisfies the Xenv interface that every tree-sitter wasm
// module imports. Across grammars these imports are identical (memory
// plus a few stubs we never reach at runtime) so one HostEnv works
// for every grammar without per-grammar wrappers.
type HostEnv struct {
	Mem *Memory
}

// Xmemory returns the host-backed linear memory to the wasm module.
func (h *HostEnv) Xmemory() MemoryAPI { return h.Mem }

// Xts_language_is_wasm: every grammar is statically linked into its wasm
// module, not loaded from a runtime .wasm via wasm_store. Always false.
func (*HostEnv) Xts_language_is_wasm(int32) int32 { return 0 }

// Xts_wasm_store_delete: unreachable, never called when is_wasm is false.
func (*HostEnv) Xts_wasm_store_delete(int32) {}

// Stdio stubs. tree-sitter's parser.c references these from debug-only
// branches we never trigger (no dot_graph_file is set from our exported
// API). Returning 0 satisfies the import.
func (*HostEnv) Xfputc(int32, int32) int32                   { return 0 }
func (*HostEnv) Xvfprintf(int32, int32, int32) int32         { return 0 }
func (*HostEnv) Xvsnprintf(int32, int32, int32, int32) int32 { return 0 }
