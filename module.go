package bonsai

// Module is the subset of every tree-sitter wasm module that the bonsai
// runtime invokes during parsing. Each grammar's generated *Module
// structurally implements it. There is no interface declaration on the
// generated side. Adding a method here requires every grammar's wasm to
// export the matching tree-sitter symbol.
//
// The grammar-specific language pointer (e.g. Xtree_sitter_python) is
// NOT part of this interface. Each grammar's NewParser fetches it from
// its own *Module and passes it to NewFromModule as a plain int32.
type Module interface {
	X_initialize()

	Xmalloc(int32) int32
	Xfree(int32)

	Xts_parser_new() int32
	Xts_parser_delete(int32)
	Xts_parser_set_language(parser, language int32) int32
	Xts_parser_parse_string(parser, oldTree, src, length int32) int32

	Xts_tree_delete(int32)
	Xts_tree_root_node(sret, tree int32)

	Xts_node_type(sret int32) int32
	Xts_node_start_byte(sret int32) int32
	Xts_node_end_byte(sret int32) int32
	Xts_node_is_named(sret int32) int32
	Xts_node_is_error(sret int32) int32
	Xts_node_is_missing(sret int32) int32
	Xts_node_has_error(sret int32) int32
	Xts_node_start_point(sret, node int32)
	Xts_node_end_point(sret, node int32)

	Xts_tree_cursor_new(sret, node int32)
	Xts_tree_cursor_delete(cursor int32)
	Xts_tree_cursor_current_node(sret, cursor int32)
	Xts_tree_cursor_current_field_name(cursor int32) int32
	Xts_tree_cursor_goto_first_child(cursor int32) int32
	Xts_tree_cursor_goto_next_sibling(cursor int32) int32
	Xts_tree_cursor_goto_parent(cursor int32) int32
}
