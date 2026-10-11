package serviceioconnector

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestValidateCompleteTraceBatch_Empty(t *testing.T) {
	require.NoError(t, validateCompleteTraceBatch(ptrace.NewTraces()))
}

func TestValidateCompleteTraceBatch_SingleTrace(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()
	span := ss.Spans().AppendEmpty()
	span.SetTraceID(pcommon.TraceID([16]byte{1, 2, 3}))

	require.NoError(t, validateCompleteTraceBatch(td))
}

func TestValidateCompleteTraceBatch_MultipleTraceIDs(t *testing.T) {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	span1 := ss.Spans().AppendEmpty()
	span1.SetTraceID(pcommon.TraceID([16]byte{1}))

	span2 := ss.Spans().AppendEmpty()
	span2.SetTraceID(pcommon.TraceID([16]byte{2}))

	require.Error(t, validateCompleteTraceBatch(td))
}
