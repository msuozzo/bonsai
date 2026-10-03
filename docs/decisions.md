# Decision log

A few notes on choices we've made and why.

## No `-msimd128`

No measurable win on bonsai v0.4.0 (bonsai-go, tree-sitter v0.25.10):
tree-sitter's core is branchy table-driven state machines, so at `-Oz` clang
emits no vector lane ops at all, and even at `-O3` it vectorizes under 1% of
the parse profile. With a prototype wasm2go `simd/archsimd` build, we measured
level-to-slightly-slower (+2-6%) than the scalar version.

## `-Oz`, not `-O3`

`-O3` performs ~15% better (benchstat n=10 on bonsai-go with wasm2go v0.4.16,
all fixture sizes), but roughly doubles the generated code size and, as a
consequence, the consumer cost to compile the generated package on first
install. bonsai-go's `module_gen.go` grows 528 KB -> 920 KB, compile time 0.49
s -> 1.05 s, compiler max RSS 251 MB -> 410 MB. That cost scales with grammar size
(bash's generated file is ~3.6× go's), so we keep `-Oz` and its cheap installs.
Revisit if parse speed is a bottleneck.
