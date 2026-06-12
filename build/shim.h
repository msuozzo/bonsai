/* shim.h: injected via clang -include before every TU.
 *
 * tree-sitter's core was written against a hosted libc. Our wasm-side libc
 * (from libc-gen) is freestanding-minimal: a handful of symbols are missing.
 * Most are referenced only from dead branches (debug logging, parser
 * timeouts), but the compiler still type-checks them. Provide the
 * declarations here. Matching definitions live in build/stubs.c.
 */
#pragma once

/* time.h adjuncts: clock.h needs CLOCKS_PER_SEC and clock(). We never enable
 * parser timeouts, but the values are referenced from inline helpers. */
#ifndef CLOCKS_PER_SEC
#define CLOCKS_PER_SEC 1000000
#endif
typedef long ts_shim_clock_t;
ts_shim_clock_t clock(void);

/* stdio.h adjuncts: LOG_STACK / LOG_TREE in parser.c reference fputs and
 * fdopen inside `if (self->dot_graph_file)` blocks. We never set
 * dot_graph_file (no public setter is exported), so the calls are
 * unreachable at runtime, but the compiler still needs prototypes. */
#include <stdio.h>
int fputs(const char *, FILE *);
FILE *fdopen(int, const char *);
