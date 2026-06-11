/* wchar.h — freestanding stub for grammar scanners (same approach as
 * ../shim.h). Scanners include it for the wide-char types; the wide
 * string functions are deliberately absent so accidental use fails the
 * build instead of misbehaving at runtime. */
#pragma once

#include <stddef.h> /* wchar_t (clang freestanding builtin header) */

typedef unsigned int wint_t;

#ifndef WEOF
#define WEOF 0xffffffffu
#endif
