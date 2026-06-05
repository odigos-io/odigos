package netmetrics

import "sort"

// CallHop is one node in a downstream call path, flattened depth-first with its depth
// so callers can render it as an indented tree (a request path), not a node-link graph.
type CallHop struct {
	Name        string    `json:"name"`
	Depth       int       `json:"depth"`         // 0 = the root
	State       NodeState `json:"state"`         // instrumented / discovered / external
	BytesPerSec float64   `json:"bytes_per_sec"` // throughput of this node
	Last        bool      `json:"last"`          // last child of its parent (for tree glyphs)
	Cycle       bool      `json:"cycle"`         // already visited on this path — not expanded
}

// CallTree is a node's end-to-end downstream request path plus trace-coverage analysis.
// It answers two questions at once: "what does a request through this node touch?" and
// "what must I instrument to trace it end to end?".
type CallTree struct {
	Root string    `json:"root"`
	Hops []CallHop `json:"hops"` // DFS-ordered, root first; render indented by Depth

	// Coverage over the DISTINCT downstream services (excluding external peers, which
	// can't be instrumented here): how many are already traced and which are the gaps.
	Total        int      `json:"total"`        // distinct instrumentable hops in the path
	Instrumented int      `json:"instrumented"` // of Total, how many are ●
	Gaps         []string `json:"gaps"`         // ◐ discovered hops, in path order — instrument these
}

// BuildCallTree walks the downstream call graph from root over ServiceNode.Out, producing
// an indented request path. Cycles terminate (a node already on the current path is marked
// Cycle and not re-expanded); a node reached by multiple paths is shown once (first reached).
// External peers are leaves (they have no instrumentable downstream here). maxDepth bounds
// runaway depth (0 = a sensible default).
func BuildCallTree(snap Snapshot, root string, maxDepth int) CallTree {
	if maxDepth <= 0 {
		maxDepth = 16
	}
	byName := make(map[string]ServiceNode, len(snap.Nodes))
	for _, n := range snap.Nodes {
		byName[n.Name] = n
	}

	ct := CallTree{Root: root}
	visited := map[string]bool{} // nodes already emitted (dedupe across branches)
	onPath := map[string]bool{}  // nodes on the current DFS stack (cycle detection)

	var walk func(name string, depth int, last bool)
	walk = func(name string, depth int, last bool) {
		n, ok := byName[name]
		hop := CallHop{Name: name, Depth: depth, Last: last}
		if ok {
			hop.State, hop.BytesPerSec = n.State, n.BytesPerSec
		} else {
			hop.State = StateExternal
		}
		if onPath[name] {
			hop.Cycle = true
			ct.Hops = append(ct.Hops, hop)
			return
		}
		ct.Hops = append(ct.Hops, hop)
		visited[name] = true

		// coverage: count distinct instrumentable (non-external) downstream hops.
		if depth > 0 && hop.State != StateExternal {
			ct.Total++
			if hop.State == StateInstrumented {
				ct.Instrumented++
			} else {
				ct.Gaps = append(ct.Gaps, name)
			}
		}

		if depth >= maxDepth || !ok || len(n.Out) == 0 {
			return
		}
		onPath[name] = true
		// stable child order: by throughput desc then name, so the path reads hottest-first.
		children := append([]string(nil), n.Out...)
		sort.SliceStable(children, func(i, j int) bool {
			ci, cj := byName[children[i]], byName[children[j]]
			if ci.BytesPerSec != cj.BytesPerSec {
				return ci.BytesPerSec > cj.BytesPerSec
			}
			return children[i] < children[j]
		})
		// skip children already emitted elsewhere to keep the tree finite and readable.
		var todo []string
		for _, c := range children {
			if !visited[c] || onPath[c] {
				todo = append(todo, c)
			}
		}
		for i, c := range todo {
			walk(c, depth+1, i == len(todo)-1)
		}
		onPath[name] = false
	}

	walk(root, 0, true)
	return ct
}
