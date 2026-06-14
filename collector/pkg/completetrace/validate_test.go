package completetrace

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestValidateCompleteTrace_Empty(t *testing.T) {
	_, shouldProcess, spanCount, err := ValidateCompleteTrace(ptrace.NewTraces())
	require.NoError(t, err)
	require.False(t, shouldProcess)
	require.Equal(t, 0, spanCount)
}

func TestValidateCompleteTrace_SingleTrace(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	traceID := pcommon.TraceID([16]byte{1, 2, 3})
	span.SetTraceID(traceID)

	gotTraceID, shouldProcess, spanCount, err := ValidateCompleteTrace(td)
	require.NoError(t, err)
	require.True(t, shouldProcess)
	require.Equal(t, 1, spanCount)
	require.Equal(t, traceID, gotTraceID)
}

func TestValidateCompleteTrace_MultipleTraceIDs(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	span1 := ss.Spans().AppendEmpty()
	span1.SetTraceID(pcommon.TraceID([16]byte{1}))

	span2 := ss.Spans().AppendEmpty()
	span2.SetTraceID(pcommon.TraceID([16]byte{2}))

	_, shouldProcess, spanCount, err := ValidateCompleteTrace(td)
	require.Error(t, err)
	require.False(t, shouldProcess)
	require.Equal(t, 0, spanCount)
}

func TestValidateCompleteTrace_OdigosTraceState(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.TraceState().FromRaw("odigos=sampled")

	_, shouldProcess, spanCount, err := ValidateCompleteTrace(td)
	require.NoError(t, err)
	require.False(t, shouldProcess)
	require.Equal(t, 0, spanCount)
}

func TestExtractOdigosTraceStateValue(t *testing.T) {
	require.Equal(t, "sampled", ExtractOdigosTraceStateValue("odigos=sampled,other=value"))
	require.Equal(t, "", ExtractOdigosTraceStateValue("other=value"))
}
