/* stubs.c: definitions for symbols tree-sitter references but never reaches
 * at runtime in the bonsai build configuration. Declarations live in shim.h and
 * (for the wasm_store surface) in tree-sitter/lib/src/wasm_store.h.
 */
#include <stddef.h>
#include <stdint.h>
#include <stdbool.h>

#include "tree_sitter/api.h"
#include "wasm_store.h"

/* dlmalloc-derived allocator. Pulled in as a header (not a separate TU) so
 * the `static` init_allocator() is visible to _initialize below. The build
 * script does NOT pass libc/malloc_sbrk.c to clang separately. */
#include "malloc_sbrk.c"

/* Reactor entry point. wasm-ld in -mexec-model=reactor expects an exported
 * `_initialize` that runs global constructors via __wasm_call_ctors and
 * any libc bootstrap. init_allocator hands the dlmalloc-derived allocator
 * the initial linear-memory heap. Without it, calls to malloc fall back
 * to sbrk-based growth (slower path, but still correct). */
extern void __wasm_call_ctors(void);
__attribute__((export_name("_initialize")))
void _initialize(void) {
	__wasm_call_ctors();
	init_allocator();
}

/* time.h: parser.c includes clock.h for its built-in timeout logic. We never
 * set a timeout (no ts_parser_set_timeout_micros export), but clock() is
 * referenced from an inline helper, so a definition is required. */
long clock(void) { return 0; }

/* stdio.h: dead-code-eliminated LOG_STACK / LOG_TREE calls.
 * Trivial stubs. wasm-opt and the linker eliminate the call sites. */
int fputs(const char *s, FILE *f) { (void)s; (void)f; return 0; }
FILE *fdopen(int fd, const char *mode) { (void)fd; (void)mode; return NULL; }

/* tree-sitter wasm_store API: only invoked when language->is_wasm is true.
 * Each grammar is linked natively into its module (the module exports its
 * own tree_sitter_<lang> symbol), so is_wasm is always false. Stubs satisfy
 * the linker. */
bool ts_wasm_store_start(TSWasmStore *self, TSLexer *lexer, const TSLanguage *language) {
	(void)self; (void)lexer; (void)language;
	return false;
}
void ts_wasm_store_reset(TSWasmStore *self) { (void)self; }
bool ts_wasm_store_has_error(const TSWasmStore *self) { (void)self; return false; }

bool ts_wasm_store_call_lex_main(TSWasmStore *self, TSStateId state) {
	(void)self; (void)state;
	return false;
}
bool ts_wasm_store_call_lex_keyword(TSWasmStore *self, TSStateId state) {
	(void)self; (void)state;
	return false;
}

uint32_t ts_wasm_store_call_scanner_create(TSWasmStore *self) { (void)self; return 0; }
void ts_wasm_store_call_scanner_destroy(TSWasmStore *self, uint32_t scanner_address) {
	(void)self; (void)scanner_address;
}
bool ts_wasm_store_call_scanner_scan(TSWasmStore *self, uint32_t scanner_address, uint32_t valid_tokens_ix) {
	(void)self; (void)scanner_address; (void)valid_tokens_ix;
	return false;
}
uint32_t ts_wasm_store_call_scanner_serialize(TSWasmStore *self, uint32_t scanner_address, char *buffer) {
	(void)self; (void)scanner_address; (void)buffer;
	return 0;
}
void ts_wasm_store_call_scanner_deserialize(TSWasmStore *self, uint32_t scanner, const char *buffer, unsigned length) {
	(void)self; (void)scanner; (void)buffer; (void)length;
}

void ts_wasm_language_retain(const TSLanguage *self) { (void)self; }
void ts_wasm_language_release(const TSLanguage *self) { (void)self; }
