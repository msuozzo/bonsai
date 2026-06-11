/* wctype.h — freestanding stub for grammar scanners.
 *
 * Scanners call these on tree-sitter lookahead code points. ASCII gets
 * correct C-locale classification; beyond ASCII, iswspace knows the
 * Unicode White_Space code points and everything else classifies as
 * "no" (towlower/towupper are identity). That is narrower than a hosted
 * libc's full-Unicode tables, which is acceptable for the scanners we
 * currently build — they use these for ASCII-defined syntax (HTML tag
 * names, list markers, heredoc delimiters). Revisit with real tables if
 * a grammar ever keys real Unicode classes off these.
 */
#pragma once

typedef unsigned int wint_t;

static inline int iswspace(wint_t c) {
	switch (c) {
	case 0x09: case 0x0a: case 0x0b: case 0x0c: case 0x0d: case 0x20:
	case 0x85: case 0xa0: case 0x1680:
	case 0x2000: case 0x2001: case 0x2002: case 0x2003: case 0x2004:
	case 0x2005: case 0x2006: case 0x2007: case 0x2008: case 0x2009:
	case 0x200a: case 0x2028: case 0x2029: case 0x202f: case 0x205f:
	case 0x3000:
		return 1;
	default:
		return 0;
	}
}

static inline int iswdigit(wint_t c) { return c >= '0' && c <= '9'; }
static inline int iswlower(wint_t c) { return c >= 'a' && c <= 'z'; }
static inline int iswupper(wint_t c) { return c >= 'A' && c <= 'Z'; }
static inline int iswalpha(wint_t c) { return iswlower(c) || iswupper(c); }
static inline int iswalnum(wint_t c) { return iswalpha(c) || iswdigit(c); }
static inline int iswblank(wint_t c) { return c == ' ' || c == '\t'; }

static inline int iswxdigit(wint_t c) {
	return iswdigit(c) || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F');
}

static inline wint_t towlower(wint_t c) { return iswupper(c) ? c + 32 : c; }
static inline wint_t towupper(wint_t c) { return iswlower(c) ? c - 32 : c; }
