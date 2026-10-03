package flamegraph

import (
	"fmt"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dumpTree renders the tree in child order so tests can assert the sibling ordering that
// InsertStack's binary search depends on, not just the aggregated totals.
func dumpTree(t *Tree) []string {
	var out []string
	var walk func(n *node, depth int)
	walk = func(n *node, depth int) {
		out = append(out, fmt.Sprintf("%s%s self=%d total=%d", strings.Repeat(" ", depth), n.name, n.self, n.total))
		for _, c := range n.children {
			walk(c, depth+1)
		}
	}
	for _, r := range t.root {
		walk(r, 0)
	}
	return out
}

func childNames(n []*node) []string {
	out := make([]string, 0, len(n))
	for _, c := range n {
		out = append(out, c.name)
	}
	return out
}

func TestInsertStackBuildsSelfAndTotalWeights(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(3, "main", "handler", "read")
	tr.InsertStack(7, "main", "handler", "write")

	assert.Equal(t, []string{
		"main self=0 total=10",
		" handler self=0 total=10",
		"  read self=3 total=3",
		"  write self=7 total=7",
	}, dumpTree(tr))
}

func TestInsertStackAccumulatesSelfWeightOnTheLeafOnly(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(2, "main", "handler")
	tr.InsertStack(5, "main")

	assert.Equal(t, []string{
		"main self=5 total=7",
		" handler self=2 total=2",
	}, dumpTree(tr))
}

func TestInsertStackIgnoresNonPositiveValues(t *testing.T) {
	for _, value := range []int64{0, -1, -1000} {
		t.Run(fmt.Sprintf("value_%d", value), func(t *testing.T) {
			tr := NewTree()
			tr.InsertStack(value, "main")
			assert.Empty(t, dumpTree(tr))
			assert.Empty(t, tr.AggregateSymbolStats())
		})
	}
}

func TestInsertStackIgnoresAnEmptyStack(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(5)
	assert.Empty(t, dumpTree(tr))
}

func TestInsertStackDropsAStackMadeEntirelyOfEmptyFrameNames(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(5, "", "")
	assert.Empty(t, dumpTree(tr))
}

func TestInsertStackSkipsEmptyFrameNamesInTheMiddleOfAStack(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(4, "main", "", "leaf")

	assert.Equal(t, []string{
		"main self=0 total=4",
		" leaf self=4 total=4",
	}, dumpTree(tr))
}

// The children slice is kept sorted so the sort.Search lookup in InsertStack finds an existing
// sibling instead of appending a duplicate node for the same frame.
func TestInsertStackKeepsSiblingsSortedRegardlessOfInsertionOrder(t *testing.T) {
	tr := NewTree()
	for _, frame := range []string{"zulu", "alpha", "mike", "bravo"} {
		tr.InsertStack(1, "main", frame)
	}

	require.Len(t, tr.root, 1)
	assert.Equal(t, []string{"alpha", "bravo", "mike", "zulu"}, childNames(tr.root[0].children))
}

func TestInsertStackMergesARepeatedFrameInsteadOfAppendingADuplicateSibling(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(1, "main", "zulu")
	tr.InsertStack(1, "main", "alpha")
	tr.InsertStack(5, "main", "zulu")

	require.Len(t, tr.root, 1)
	assert.Equal(t, []string{"alpha", "zulu"}, childNames(tr.root[0].children))
	assert.Equal(t, []string{
		"main self=0 total=7",
		" alpha self=1 total=1",
		" zulu self=6 total=6",
	}, dumpTree(tr))
}

func TestInsertStackKeepsRootLevelSiblingsSorted(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(1, "zulu")
	tr.InsertStack(1, "alpha")
	tr.InsertStack(1, "mike")

	assert.Equal(t, []string{"alpha", "mike", "zulu"}, childNames(tr.root))
}

func TestInsertStackTreatsRecursionAsDistinctNodes(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(4, "recurse", "recurse")

	assert.Equal(t, []string{
		"recurse self=0 total=4",
		" recurse self=4 total=4",
	}, dumpTree(tr))
}

func TestAggregateSymbolStatsSumsEveryNodeSharingAFrameName(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(3, "main", "shared", "leafA")
	tr.InsertStack(5, "second", "shared", "leafB")
	tr.InsertStack(2, "main", "shared")

	stats := tr.AggregateSymbolStats()
	byName := make(map[string]SymbolStats, len(stats))
	for _, s := range stats {
		byName[s.Name] = s
	}

	assert.Equal(t, SymbolStats{Name: "shared", Self: 2, Total: 10}, byName["shared"])
	assert.Equal(t, SymbolStats{Name: "main", Self: 0, Total: 5}, byName["main"])
	assert.Equal(t, SymbolStats{Name: "second", Self: 0, Total: 5}, byName["second"])
	assert.Equal(t, SymbolStats{Name: "leafA", Self: 3, Total: 3}, byName["leafA"])
	assert.Equal(t, SymbolStats{Name: "leafB", Self: 5, Total: 5}, byName["leafB"])
}

func TestAggregateSymbolStatsSortsBySelfDescendingThenNameAscending(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(9, "big")
	tr.InsertStack(4, "zebra")
	tr.InsertStack(4, "apple")
	tr.InsertStack(1, "small")

	stats := tr.AggregateSymbolStats()
	names := make([]string, 0, len(stats))
	for _, s := range stats {
		names = append(names, s.Name)
	}
	assert.Equal(t, []string{"big", "apple", "zebra", "small"}, names)
}

// "total" is the synthetic root bar and "other" is the truncation bucket; neither is a real
// symbol, so they must not appear as rows in the top table.
func TestAggregateSymbolStatsExcludesTheSyntheticRootAndOtherBars(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(5, "total")
	tr.InsertStack(6, otherName)
	tr.InsertStack(7, "realFrame")

	stats := tr.AggregateSymbolStats()
	require.Len(t, stats, 1)
	assert.Equal(t, "realFrame", stats[0].Name)
}

func TestAggregateSymbolStatsStillDescendsThroughAnExcludedFrame(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(5, "total", "realFrame")

	stats := tr.AggregateSymbolStats()
	require.Len(t, stats, 1)
	assert.Equal(t, SymbolStats{Name: "realFrame", Self: 5, Total: 5}, stats[0])
}

func TestAggregateSymbolStatsOnANilTreeReturnsNothing(t *testing.T) {
	var tr *Tree
	assert.Nil(t, tr.AggregateSymbolStats())
}

func TestAggregateSymbolStatsOnAnEmptyTreeReturnsNoRows(t *testing.T) {
	assert.Empty(t, NewTree().AggregateSymbolStats())
}

func TestAggregateSymbolStatsTotalsMatchTheInsertedWeight(t *testing.T) {
	tr := NewTree()
	tr.InsertStack(3, "main", "a")
	tr.InsertStack(4, "main", "b")

	stats := tr.AggregateSymbolStats()
	sort.Slice(stats, func(i, j int) bool { return stats[i].Name < stats[j].Name })

	var totalSelf int64
	for _, s := range stats {
		totalSelf += s.Self
	}
	assert.Equal(t, int64(7), totalSelf)
}
