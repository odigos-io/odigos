package flamegraph

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// exampleStackHTTPLatency is a root-first call chain aligned with
// testdata/minimal_otlp_profile.json (synthetic symbols, no real workload data).
var exampleStackHTTPLatency = []string{"runtime_main", "net_http_serve", "handler_process"}

func TestInsertStack_IgnoresNonPositiveOrEmpty(t *testing.T) {
	t.Parallel()
	tr := NewTree()
	tr.InsertStack(0, exampleStackHTTPLatency...)
	tr.InsertStack(-1, exampleStackHTTPLatency...)
	tr.InsertStack(1)
	fb := TreeToFlamebearer(tr, 0)
	assert.Equal(t, int64(0), fb.NumTicks)
}

func TestInsertStack_MergesDuplicatePaths(t *testing.T) {
	t.Parallel()
	tr := NewTree()
	tr.InsertStack(2, exampleStackHTTPLatency...)
	tr.InsertStack(3, exampleStackHTTPLatency...)
	fb := TreeToFlamebearer(tr, 0)
	assert.Equal(t, int64(5), fb.NumTicks)
	assert.Equal(t, int64(5), fb.MaxSelf)
}

func TestInsertStack_BranchesUnderSharedPrefix(t *testing.T) {
	t.Parallel()
	tr := NewTree()
	tr.InsertStack(2, exampleStackHTTPLatency...)
	tr.InsertStack(3, exampleStackHTTPLatency...)
	// Same outer frames as the example, different leaf (branching handler).
	tr.InsertStack(1, "runtime_main", "net_http_serve", "handler_alt")
	fb := TreeToFlamebearer(tr, 0)
	assert.Equal(t, int64(6), fb.NumTicks)
	assert.Contains(t, fb.Names, "handler_process")
	assert.Contains(t, fb.Names, "handler_alt")
}

func TestInsertStack_SkipsEmptyFrameNames(t *testing.T) {
	t.Parallel()
	tr := NewTree()
	tr.InsertStack(4, "runtime_main", "", "handler_process")
	fb := TreeToFlamebearer(tr, 0)
	assert.Equal(t, int64(4), fb.NumTicks)
	assert.Contains(t, fb.Names, "runtime_main")
	assert.Contains(t, fb.Names, "handler_process")
}
