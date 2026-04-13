package flamegraph

import fgminheap "github.com/grafana/pyroscope/pkg/util/minheap"

type Flamebearer struct {
	Names    []string  `json:"names"`
	Levels   [][]int64 `json:"levels"`
	NumTicks int64     `json:"numTicks"`
	MaxSelf  int64     `json:"maxSelf"`
}

// FlamebearerProfile is the full response
type FlamebearerProfile struct {
	Version     int                  `json:"version"`
	Flamebearer Flamebearer          `json:"flamebearer"`
	Metadata    FlamebearerMetadata  `json:"metadata"`
	Timeline    *FlamebearerTimeline `json:"timeline,omitempty"`
	Groups      interface{}          `json:"groups"` // null for single-profile payloads
	Heatmap     interface{}          `json:"heatmap"`
	Symbols     []SymbolStats        `json:"symbols,omitempty"`
}

// FlamebearerMetadata describes the profile
type FlamebearerMetadata struct {
	Format      string `json:"format"`                // "single"
	SpyName     string `json:"spyName"`               // e.g. "ebpf" or ""
	SampleRate  int    `json:"sampleRate"`            // e.g. 1000000000 (Hz) or 0
	Units       string `json:"units"`                 // e.g. "samples"
	Name        string `json:"name"`                  // e.g. "cpu"
	SymbolsHint string `json:"symbolsHint,omitempty"` // Shown in UI when symbols are placeholders (frame_N)
}

// FlamebearerTimeline is optional timeline data
type FlamebearerTimeline struct {
	StartTime     int64   `json:"startTime"`
	Samples       []int64 `json:"samples"`
	DurationDelta int     `json:"durationDelta"`
	Watermarks    *[]int  `json:"watermarks"` // null for single-profile payloads
}

const (
	defaultMaxNodes = 1024
	otherName       = "other"
)

// TreeToFlamebearer encodes a merged Tree into flame-bearer JSON fields (names, levels,
// numTicks, maxSelf). Small branches fold into "other" when the tree exceeds maxNodes (default 1024).
func TreeToFlamebearer(t *Tree, maxNodes int64) Flamebearer {
	if maxNodes <= 0 {
		maxNodes = defaultMaxNodes
	}
	var total, maxSelf int64
	for _, n := range t.root {
		total += n.total
	}
	names := []string{}
	nameIdx := map[string]int{}
	var levels [][]int64
	minVal := t.minValue(maxNodes)

	type item struct {
		xOffset int64
		level   int
		n       *node
	}
	stack := []item{{0, 0, &node{children: t.root, total: total}}}

	for len(stack) > 0 {
		cur := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		n := cur.n
		if n.self > maxSelf {
			maxSelf = n.self
		}
		name := n.name
		if name == "" && cur.level == 0 {
			name = "total"
		}
		idx, ok := nameIdx[name]
		if !ok {
			idx = len(names)
			nameIdx[name] = idx
			names = append(names, name)
		}
		for cur.level >= len(levels) {
			levels = append(levels, []int64{})
		}
		row := levels[cur.level]
		// Append: xOffset (will delta-encode later), total, self, nameIndex
		row = append(row, cur.xOffset, n.total, n.self, int64(idx))
		levels[cur.level] = row

		// Compute xOffset per child left-to-right; push in reverse so pop order = left-to-right.
		xStart := cur.xOffset
		var otherTotal int64
		offsets := make([]int64, len(n.children))
		for i := 0; i < len(n.children); i++ {
			c := n.children[i]
			if c.total >= minVal && c.name != otherName {
				offsets[i] = xStart
				xStart += c.total
			} else {
				otherTotal += c.total
			}
		}
		for i := len(n.children) - 1; i >= 0; i-- {
			c := n.children[i]
			if c.total >= minVal && c.name != otherName {
				stack = append(stack, item{xOffset: offsets[i], level: cur.level + 1, n: c})
			}
		}
		if otherTotal > 0 {
			stack = append(stack, item{xOffset: xStart, level: cur.level + 1, n: &node{name: otherName, self: otherTotal, total: otherTotal}})
		}
	}

	// Delta-encode x offsets (first of each 4-tuple)
	for _, row := range levels {
		var prev int64
		for i := 0; i < len(row); i += 4 {
			row[i] -= prev
			prev += row[i] + row[i+1]
		}
	}

	return Flamebearer{
		Names:    names,
		Levels:   levels,
		NumTicks: total,
		MaxSelf:  maxSelf,
	}
}

// minValue returns the minimum node total to include (nodes below are folded into "other").
// Uses a min-heap over node totals (same idea as common flame-graph folding implementations).
func (t *Tree) minValue(maxNodes int64) int64 {
	if maxNodes < 1 {
		return 0
	}
	const defaultDFSSize = 128
	nodes := make([]*node, 0, defaultDFSSize)
	for _, r := range t.root {
		nodes = append(nodes, r)
	}
	var n *node
	h := make([]int64, 0, maxNodes+1)
	for len(nodes) > 0 {
		last := len(nodes) - 1
		n, nodes = nodes[last], nodes[:last]
		if len(h) >= int(maxNodes) {
			if n.total > h[0] {
				h = fgminheap.Pop(h)
			} else {
				continue
			}
		}
		h = fgminheap.Push(h, n.total)
		nodes = append(nodes, n.children...)
	}
	if int64(len(h)) < maxNodes {
		return 0
	}
	return h[0]
}
