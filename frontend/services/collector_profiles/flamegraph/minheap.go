// Package flamegraph: min-heap for Pyroscope-compatible minValue (node folding).
// Logic from Grafana Pyroscope pkg/util/minheap.

package flamegraph

func minHeapPush(h []int64, x int64) []int64 {
	h = append(h, x)
	minHeapUp(h, len(h)-1)
	return h
}

func minHeapPop(h []int64) []int64 {
	n := len(h) - 1
	h[0], h[n] = h[n], h[0]
	minHeapDown(h, 0, n)
	return h[0 : n-1]
}

func minHeapUp(h []int64, j int) {
	for {
		i := (j - 1) / 2
		if i == j || h[j] >= h[i] {
			break
		}
		h[i], h[j] = h[j], h[i]
		j = i
	}
}

func minHeapDown(h []int64, i0, n int) {
	i := i0
	for {
		j1 := 2*i + 1
		if j1 >= n || j1 < 0 {
			break
		}
		j := j1
		if j2 := j1 + 1; j2 < n && h[j2] < h[j1] {
			j = j2
		}
		if h[j] >= h[i] {
			break
		}
		h[i], h[j] = h[j], h[i]
		i = j
	}
}
