package flamegraph

import "sort"

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

// Total returns the sum of all root children totals.
func (t *Tree) Total() int64 {
	var v int64
	for _, n := range t.root {
		v += n.total
	}
	return v
}

// SymbolTable aggregates self/total per symbol name from the tree, sorted by self descending.
func (t *Tree) SymbolTable() []SymbolStats {
	byName := make(map[string]*SymbolStats)
	var visit func(*node)
	visit = func(n *node) {
		if n == nil || n.name == "" {
			return
		}
		if s, ok := byName[n.name]; ok {
			s.Self += n.self
			s.Total += n.total
		} else {
			byName[n.name] = &SymbolStats{Name: n.name, Self: n.self, Total: n.total}
		}
		for _, c := range n.children {
			visit(c)
		}
	}
	for _, r := range t.root {
		visit(r)
	}
	out := make([]SymbolStats, 0, len(byName))
	for _, s := range byName {
		out = append(out, *s)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Self > out[j].Self })
	return out
}
