// Package bench compares bonsai-go (wasm2go-translated tree-sitter) with
// github.com/tree-sitter/go-tree-sitter (cgo bindings to the native
// tree-sitter library) on two workloads that bracket real use:
//
//   - Check parses a file and reports whether it has syntax errors, as a CI
//     gate or pre-commit hook would. It reads one node, so cgo, which leaves
//     the tree in C and answers questions on demand, does the least work
//     possible. bonsai always copies the whole tree into Go first.
//   - Export parses a file and writes the whole tree as JSON (kind, field,
//     byte range, and children for every node), as a tool handing an AST
//     to another process would. It reads every node, so cgo pays several
//     calls per node while bonsai reads Go fields.
//
// Both sides run the same grammar (tree-sitter-go), reuse one parser across
// iterations, and free trees after use. The cgo side walks with a tree
// cursor and looks names up by numeric kind and field id, its fastest path.
// TestSameResults checks that both write byte-identical JSON.
//
// The cgo dependency lives in this submodule rather than in bonsai's main
// go.mod so anyone importing bonsai or its grammar modules never inherits a
// cgo build requirement.
package bench

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"strconv"
	"testing"

	bonsaigo "github.com/msuozzo/bonsai/bonsai-go"
	tree_sitter "github.com/tree-sitter/go-tree-sitter"
	tree_sitter_go "github.com/tree-sitter/tree-sitter-go/bindings/go"
)

//go:embed testdata/tiny.go
var tinySrc []byte

//go:embed testdata/small.go
var smallSrc []byte

//go:embed testdata/large.go
var largeSrc []byte

var fixtures = []struct {
	name string
	src  []byte
}{
	{"tiny", tinySrc},
	{"small", smallSrc},
	{"large", largeSrc},
}

// lib is one way to parse Go from Go. Its funcs share a parser created up
// front and reused across calls.
type lib struct {
	name   string
	check  func(src []byte) (hasError bool)
	export func(src, out []byte) []byte
	close  func()
}

var libs = []func() lib{newBonsai, newCgo}

// quote returns s as a JSON string literal.
func quote(s string) []byte {
	b, _ := json.Marshal(s)
	return b
}

// appendNode appends the opening of a node's JSON object: its kind, its
// field when non-nil, and its byte range. kind and field are JSON-quoted.
// The caller appends any children and the closing brace.
func appendNode(out, kind, field []byte, start, end uint32) []byte {
	out = append(out, `{"kind":`...)
	out = append(out, kind...)
	if field != nil {
		out = append(out, `,"field":`...)
		out = append(out, field...)
	}
	out = append(out, `,"start":`...)
	out = strconv.AppendUint(out, uint64(start), 10)
	out = append(out, `,"end":`...)
	return strconv.AppendUint(out, uint64(end), 10)
}

// newBonsai returns a lib backed by bonsai-go. Parse materializes the whole
// tree as Go values, so exporting it reads plain struct fields.
func newBonsai() lib {
	p := bonsaigo.NewParser()
	parse := func(src []byte) *bonsaigo.Node {
		root, err := p.Parse(src)
		if err != nil {
			panic(err)
		}
		return root
	}
	// bonsai exposes kinds and fields only as strings, so quote each one on
	// first use, the counterpart to the cgo side's id-indexed tables.
	quoted := map[string][]byte{}
	name := func(s string) []byte {
		q, ok := quoted[s]
		if !ok {
			q = quote(s)
			quoted[s] = q
		}
		return q
	}
	var walk func(n *bonsaigo.Node, out []byte) []byte
	walk = func(n *bonsaigo.Node, out []byte) []byte {
		var field []byte
		if n.Field != "" {
			field = name(n.Field)
		}
		out = appendNode(out, name(n.Kind), field, n.StartByte, n.EndByte)
		if len(n.Children) > 0 {
			out = append(out, `,"children":[`...)
			for i, ch := range n.Children {
				if i > 0 {
					out = append(out, ',')
				}
				out = walk(ch, out)
			}
			out = append(out, ']')
		}
		return append(out, '}')
	}
	return lib{
		name:   "bonsai",
		check:  func(src []byte) bool { return parse(src).HasError() },
		export: func(src, out []byte) []byte { return walk(parse(src), out) },
		close:  func() {},
	}
}

// newCgo returns a lib backed by the official cgo binding. The tree stays
// in C, so every node read is a cgo call.
func newCgo() lib {
	lang := tree_sitter.NewLanguage(tree_sitter_go.Language())
	p := tree_sitter.NewParser()
	if err := p.SetLanguage(lang); err != nil {
		panic(err)
	}
	// JSON-quoted names indexed by id. Field ids start at 1, so fields[0]
	// stays nil and a node without a field writes none.
	kinds := make([][]byte, lang.NodeKindCount())
	for id := range kinds {
		kinds[id] = quote(lang.NodeKindForId(uint16(id)))
	}
	fields := make([][]byte, lang.FieldCount()+1)
	for id := 1; id < len(fields); id++ {
		fields[id] = quote(lang.FieldNameForId(uint16(id)))
	}
	var walk func(c *tree_sitter.TreeCursor, out []byte) []byte
	walk = func(c *tree_sitter.TreeCursor, out []byte) []byte {
		n := c.Node()
		out = appendNode(out, kinds[n.KindId()], fields[c.FieldId()],
			uint32(n.StartByte()), uint32(n.EndByte()))
		if c.GotoFirstChild() {
			out = append(out, `,"children":[`...)
			out = walk(c, out)
			for c.GotoNextSibling() {
				out = append(out, ',')
				out = walk(c, out)
			}
			c.GotoParent()
			out = append(out, ']')
		}
		return append(out, '}')
	}
	return lib{
		name: "cgo",
		check: func(src []byte) bool {
			tree := p.Parse(src, nil)
			defer tree.Close()
			return tree.RootNode().HasError()
		},
		export: func(src, out []byte) []byte {
			tree := p.Parse(src, nil)
			defer tree.Close()
			c := tree.Walk()
			defer c.Close()
			return walk(c, out)
		},
		close: p.Close,
	}
}

// TestSameResults checks that every lib sees the same syntax errors and
// writes byte-identical, valid JSON, so the benchmarks compare equal work.
func TestSameResults(t *testing.T) {
	for _, f := range fixtures {
		var want []byte
		var wantFrom string
		for _, newLib := range libs {
			l := newLib()
			if l.check(f.src) {
				t.Errorf("%s/%s: unexpected syntax error", l.name, f.name)
			}
			got := l.export(f.src, nil)
			l.close()
			if !json.Valid(got) {
				t.Fatalf("%s/%s: invalid JSON", l.name, f.name)
			}
			if want == nil {
				want, wantFrom = got, l.name
			} else if !bytes.Equal(got, want) {
				t.Errorf("%s/%s: JSON differs from %s's (%d vs %d bytes)",
					l.name, f.name, wantFrom, len(got), len(want))
			}
		}
	}
}

func BenchmarkCheck(b *testing.B) {
	for _, newLib := range libs {
		l := newLib()
		b.Run("lib="+l.name, func(b *testing.B) {
			for _, f := range fixtures {
				b.Run("src="+f.name, func(b *testing.B) {
					b.SetBytes(int64(len(f.src)))
					for b.Loop() {
						if l.check(f.src) {
							b.Fatal("unexpected syntax error")
						}
					}
				})
			}
		})
		l.close()
	}
}

// BenchmarkExport reuses the output buffer across iterations.
func BenchmarkExport(b *testing.B) {
	for _, newLib := range libs {
		l := newLib()
		b.Run("lib="+l.name, func(b *testing.B) {
			for _, f := range fixtures {
				b.Run("src="+f.name, func(b *testing.B) {
					b.SetBytes(int64(len(f.src)))
					var out []byte
					for b.Loop() {
						if out = l.export(f.src, out[:0]); len(out) == 0 {
							b.Fatal("no output")
						}
					}
				})
			}
		})
		l.close()
	}
}
