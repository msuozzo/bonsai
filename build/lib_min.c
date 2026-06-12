/* lib_min.c: minimal amalgamation for the read-only bonsai build.
 *
 * tree-sitter ships lib.c which #includes every translation unit including
 * query.c (depends on <wctype.h>) and wasm_store.c (depends on wasmtime
 * headers). We don't expose queries or runtime-WASM language loading, so
 * those are omitted. Everything else parser.c / language.c links against
 * is included here.
 */
#include "./alloc.c"
#include "./language.c"
#include "./lexer.c"
#include "./node.c"
#include "./parser.c"
#include "./stack.c"
#include "./subtree.c"
#include "./tree.c"
#include "./tree_cursor.c"
#include "./get_changed_ranges.c"
