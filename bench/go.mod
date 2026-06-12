// Benchmark-only module. Lives separately from bonsai's go.mod so the
// cgo-based github.com/tree-sitter/go-tree-sitter dependency does NOT
// propagate to anyone who imports bonsai or its grammar modules.
//
// Run with: go test -bench=. -benchmem ./...   (from this directory)
module github.com/msuozzo/bonsai/bench

go 1.25.0

require (
	github.com/msuozzo/bonsai/bonsai-go v0.0.0
	github.com/tree-sitter/go-tree-sitter v0.25.0
	github.com/tree-sitter/tree-sitter-go v0.25.0
)

require (
	github.com/mattn/go-pointer v0.0.1 // indirect
	github.com/msuozzo/bonsai v0.0.0 // indirect
)

replace (
	github.com/msuozzo/bonsai => ../
	github.com/msuozzo/bonsai/bonsai-go => ../bonsai-go
)
