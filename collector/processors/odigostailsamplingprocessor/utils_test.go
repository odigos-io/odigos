package odigostailsamplingprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

// appendSpans adds a new resource spans entry holding spansCount spans, all with the given trace id.
func appendSpans(td ptrace.Traces, traceID pcommon.TraceID, spansCount int) ptrace.ResourceSpans {
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	for i := 0; i < spansCount; i++ {
		span := ss.Spans().AppendEmpty()
		span.SetTraceID(traceID)
		span.SetSpanID(pcommon.SpanID{0, 0, 0, 0, 0, 0, 0, byte(i + 1)})
	}
	return rs
}

func TestCheckPrerequistsSpanCount(t *testing.T) {
	traceID := pcommon.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}

	tests := []struct {
		name              string
		spansPerResource  []int
		wantSpanCount     int
		wantShouldProcess bool
	}{
		{
			name:              "no spans",
			spansPerResource:  nil,
			wantSpanCount:     0,
			wantShouldProcess: false,
		},
		{
			name:              "single span",
			spansPerResource:  []int{1},
			wantSpanCount:     1,
			wantShouldProcess: true,
		},
		{
			name:              "multiple spans in one resource",
			spansPerResource:  []int{5},
			wantSpanCount:     5,
			wantShouldProcess: true,
		},
		{
			name:              "spans spread over multiple resources",
			spansPerResource:  []int{3, 2, 4},
			wantSpanCount:     9,
			wantShouldProcess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			td := ptrace.NewTraces()
			for _, count := range tt.spansPerResource {
				appendSpans(td, traceID, count)
			}

			gotTraceID, shouldProcess, spanCount, err := checkPrerequists(td)
			require.NoError(t, err)
			assert.Equal(t, tt.wantShouldProcess, shouldProcess)
			assert.Equal(t, tt.wantSpanCount, spanCount)
			if tt.wantShouldProcess {
				assert.Equal(t, traceID, gotTraceID)
			}
		})
	}
}

func TestCheckPrerequistsMultipleTraceIds(t *testing.T) {
	td := ptrace.NewTraces()
	appendSpans(td, pcommon.TraceID{1}, 2)
	appendSpans(td, pcommon.TraceID{2}, 2)

	_, shouldProcess, spanCount, err := checkPrerequists(td)
	require.Error(t, err)
	assert.False(t, shouldProcess)
	assert.Equal(t, 0, spanCount)
}

func TestCheckPrerequistsHeadSamplingAlreadyApplied(t *testing.T) {
	traceID := pcommon.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	td := ptrace.NewTraces()
	rs := appendSpans(td, traceID, 3)
	rs.ScopeSpans().At(0).Spans().At(2).TraceState().FromRaw("odigos=1")

	_, shouldProcess, spanCount, err := checkPrerequists(td)
	require.NoError(t, err)
	assert.False(t, shouldProcess)
	assert.Equal(t, 0, spanCount)
}
