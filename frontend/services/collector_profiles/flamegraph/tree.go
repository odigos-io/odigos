package flamegraph

import "sort"

// Tree is an in-memory merged call tree built from stack samples (root-first frame names).
// Pyroscope does not export a stable public type for this merge step; we mirror Grafana's
// flamegraph construction (see TreeToFlamebearer doc) after converting OTLP via their ingester.

// SymbolStats is one row for the symbol table (Pyroscope "Top Table").
type SymbolStats struct {
	Name  string `json:"name"`
	Self  int64  `json:"self"`
	Total int64  `json:"total"`
}

// Tree holds merged stack samples for flame graph. Root is implicit; root's children are in root.
type Tree struct {
	root []*node
}

type node struct {
	parent   *node
	children []*node
	self     int64
	total    int64
	name     string
}

// NewTree returns an empty tree.
func NewTree() *Tree {
	return &Tree{}
}

// InsertStack merges a stack (root-first frame names) with the given value into the tree.
func (t *Tree) InsertStack(value int64, stack ...string) {
	if value <= 0 || len(stack) == 0 {
		return
	}
	r := &node{children: t.root}
	n := r
	for _, name := range stack {
		if name == "" {
			continue
		}
		n.total += value
		i := sort.Search(len(n.children), func(j int) bool { return n.children[j].name >= name })
		if i < len(n.children) && n.children[i].name == name {
			n = n.children[i]
		} else {
			child := &node{parent: n, name: name}
			n.children = append(n.children, nil)
			copy(n.children[i+1:], n.children[i:])
			n.children[i] = child
			n = child
		}
	}
	n.total += value
	n.self += value
	t.root = r.children
}
