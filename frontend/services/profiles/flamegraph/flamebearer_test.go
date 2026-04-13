package flamegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTreeToFlamebearer_DefaultMaxNodes(t *testing.T) {
	t.Parallel()
	tr := NewTree()
	tr.InsertStack(10, exampleStackHTTPLatency...)
	fb := TreeToFlamebearer(tr, 0)
	assert.Greater(t, fb.NumTicks, int64(0))
	require.NotEmpty(t, fb.Names)
	require.NotEmpty(t, fb.Levels)
	assert.Contains(t, fb.Names, "handler_process")
}

func TestTreeToFlamebearer_LevelRowsAreQuadruples(t *testing.T) {
	t.Parallel()
	tr := NewTree()
	tr.InsertStack(3, exampleStackHTTPLatency...)
	fb := TreeToFlamebearer(tr, 0)
	for _, row := range fb.Levels {
		assert.Equal(t, 0, len(row)%4, "each level row should be xOffset,total,self,nameIdx quads")
	}
}

func TestTreeToFlamebearer_EmptyTree(t *testing.T) {
	t.Parallel()
	fb := TreeToFlamebearer(NewTree(), 100)
	assert.Equal(t, int64(0), fb.NumTicks)
	// Synthetic "total" row is still emitted for an empty merged tree.
	assert.Equal(t, []string{"total"}, fb.Names)
}
