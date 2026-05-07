package flamegraph

import (
	"sort"
)

// Tree is an in-memory merged call tree built from stack samples (root-first frame names).
// Pyroscope does not export a stable public type for this merge step; we mirror Grafana's
// flamegraph construction (see TreeToFlamebearer doc) after converting OTLP via their ingester.

// This tree is Odigos's post-merge aggregation model used to compute symbols/top-table and
// produce a stable API response shape. Pyroscope conversion/merge helpers are reused upstream,
// but they do not expose one public type that covers this contract end to end.
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
	self     int64  // exclusive weight: samples whose call-stack ends at this exact frame
	total    int64  // inclusive weight: self plus the weights of all descendant frames
	name     string // frame label displayed in the flame graph (function name, file:line, or address)
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
	for _, frameName := range stack {
		if frameName == "" {
			continue
		}
		n.total += value
		i := sort.Search(len(n.children), func(j int) bool { return n.children[j].name >= frameName })
		if i < len(n.children) && n.children[i].name == frameName {
			n = n.children[i]
		} else {
			child := &node{parent: n, name: frameName}
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

// AggregateSymbolStats builds top-table rows (name, self, total) by aggregating every tree node
// with the same frame name. Self is the sum of per-node self weights; total is the sum of per-node
// totals (inclusive weight at each distinct call-site node, not deduplicated across stacks).
func (t *Tree) AggregateSymbolStats() []SymbolStats {
	if t == nil {
		return nil
	}
	const rootBarName = "total"
	type agg struct {
		self, total int64
	}
	m := make(map[string]*agg)
	var walk func(*node)
	walk = func(n *node) {
		if n == nil {
			return
		}
		name := n.name
		if name != "" && name != rootBarName && name != otherName {
			a := m[name]
			if a == nil {
				a = &agg{}
				m[name] = a
			}
			a.self += n.self
			a.total += n.total
		}
		for _, c := range n.children {
			walk(c)
		}
	}
	for _, r := range t.root {
		walk(r)
	}
	out := make([]SymbolStats, 0, len(m))
	for name, a := range m {
		out = append(out, SymbolStats{Name: name, Self: a.self, Total: a.total})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Self != out[j].Self {
			return out[i].Self > out[j].Self
		}
		return out[i].Name < out[j].Name
	})
	return out
}
