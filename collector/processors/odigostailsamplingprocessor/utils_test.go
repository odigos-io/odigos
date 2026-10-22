package odigostailsamplingprocessor

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
)

func TestExtractOdigosTraceStateValue(t *testing.T) {
	tests := []struct {
		name       string
		traceState string
		want       string
	}{
		{
			name:       "empty trace state",
			traceState: "",
			want:       "",
		},
		{
			name:       "only the odigos entry",
			traceState: "odigos=1",
			want:       "1",
		},
		{
			name:       "odigos entry among other vendors",
			traceState: "congo=t61rcWkgMzE,odigos=head,rojo=00f067aa",
			want:       "head",
		},
		{
			name:       "no odigos entry",
			traceState: "congo=t61rcWkgMzE,rojo=00f067aa",
			want:       "",
		},
		{
			name:       "whitespace around the entry and around the separator",
			traceState: "congo=t61rcWkgMzE , odigos = head ",
			want:       "head",
		},
		{
			name:       "entry without a value separator is ignored",
			traceState: "odigos,congo=t61rcWkgMzE",
			want:       "",
		},
		{
			name:       "empty entries between separators are ignored",
			traceState: ",,odigos=head,,",
			want:       "head",
		},
		// A vendor key that merely contains "odigos" must not be mistaken for the odigos entry,
		// otherwise an unrelated vendor's tracestate would make Odigos skip tail sampling entirely.
		{
			name:       "key that only contains odigos is not the odigos entry",
			traceState: "myodigos=head,odigos2=head",
			want:       "",
		},
		{
			name:       "value containing an equals sign is kept whole",
			traceState: "odigos=a=b",
			want:       "a=b",
		},
		{
			name:       "odigos entry with an empty value is treated as absent",
			traceState: "odigos=",
			want:       "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, extractOdigosTraceStateValue(tt.traceState))
		})
	}
}

func TestGetRootSpan(t *testing.T) {
	t.Run("root span in a later resource and scope is found with its own resource", func(t *testing.T) {
		td := ptrace.NewTraces()

		childRes := td.ResourceSpans().AppendEmpty()
		childRes.Resource().Attributes().PutStr("service.name", "child-service")
		child := childRes.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		child.SetName("child")
		child.SetParentSpanID(pcommon.SpanID{1})

		rootRes := td.ResourceSpans().AppendEmpty()
		rootRes.Resource().Attributes().PutStr("service.name", "root-service")
		// the first scope of the second resource holds another child, so the root is only
		// reachable by walking into the second scope.
		otherChild := rootRes.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		otherChild.SetName("other-child")
		otherChild.SetParentSpanID(pcommon.SpanID{2})
		root := rootRes.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		root.SetName("root")

		gotSpan, gotResource, found := getRootSpan(td)
		require.True(t, found)
		assert.Equal(t, "root", gotSpan.Name())
		serviceName, ok := gotResource.Attributes().Get("service.name")
		require.True(t, ok)
		assert.Equal(t, "root-service", serviceName.Str())
	})

	t.Run("trace without a parentless span has no root span", func(t *testing.T) {
		td := ptrace.NewTraces()
		spans := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
		first := spans.AppendEmpty()
		first.SetParentSpanID(pcommon.SpanID{1})
		second := spans.AppendEmpty()
		second.SetParentSpanID(pcommon.SpanID{2})

		_, _, found := getRootSpan(td)
		assert.False(t, found)
	})

	t.Run("empty trace has no root span", func(t *testing.T) {
		_, _, found := getRootSpan(ptrace.NewTraces())
		assert.False(t, found)
	})
}

func TestCheckPrerequists(t *testing.T) {
	traceA := pcommon.TraceID{0xaa, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}
	traceB := pcommon.TraceID{0xbb, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15}

	t.Run("single span batch is processed", func(t *testing.T) {
		td := ptrace.NewTraces()
		span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		span.SetTraceID(traceA)

		traceID, shouldProcess, spanCount, err := checkPrerequists(td)
		require.NoError(t, err)
		assert.True(t, shouldProcess)
		assert.Equal(t, traceA, traceID)
		assert.Equal(t, 1, spanCount)
	})

	t.Run("empty batch is skipped", func(t *testing.T) {
		traceID, shouldProcess, _, err := checkPrerequists(ptrace.NewTraces())
		require.NoError(t, err)
		assert.False(t, shouldProcess)
		assert.Equal(t, pcommon.TraceID{}, traceID)
	})

	t.Run("resource span without any span is skipped", func(t *testing.T) {
		td := ptrace.NewTraces()
		td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty()

		_, shouldProcess, _, err := checkPrerequists(td)
		require.NoError(t, err)
		assert.False(t, shouldProcess)
	})

	t.Run("spans sharing a trace id across resources are processed", func(t *testing.T) {
		td := ptrace.NewTraces()
		for i := 0; i < 3; i++ {
			span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
			span.SetTraceID(traceA)
		}

		traceID, shouldProcess, _, err := checkPrerequists(td)
		require.NoError(t, err)
		assert.True(t, shouldProcess)
		assert.Equal(t, traceA, traceID)
	})

	// This processor must run after groupbytraceid. If it does not, sampling one batch would apply a
	// single decision to spans of unrelated traces, so the mixed batch has to be rejected.
	t.Run("spans from two traces in one batch is an error", func(t *testing.T) {
		td := ptrace.NewTraces()
		spans := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
		first := spans.AppendEmpty()
		first.SetTraceID(traceA)
		second := spans.AppendEmpty()
		second.SetTraceID(traceB)

		traceID, shouldProcess, _, err := checkPrerequists(td)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not all spans belong to the same trace")
		assert.False(t, shouldProcess)
		assert.Equal(t, pcommon.TraceID{}, traceID)
	})

	// Head sampling already made a decision for this trace, and re-applying tail sampling on top of
	// it would compound the two probabilities and drop far more than the user asked for.
	t.Run("odigos trace state on any span skips the batch", func(t *testing.T) {
		td := ptrace.NewTraces()
		spans := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans()
		plain := spans.AppendEmpty()
		plain.SetTraceID(traceA)
		headSampled := spans.AppendEmpty()
		headSampled.SetTraceID(traceA)
		headSampled.TraceState().FromRaw("congo=t61rcWkgMzE,odigos=head")

		traceID, shouldProcess, _, err := checkPrerequists(td)
		require.NoError(t, err)
		assert.False(t, shouldProcess)
		assert.Equal(t, pcommon.TraceID{}, traceID)
	})

	t.Run("trace state from another vendor does not skip the batch", func(t *testing.T) {
		td := ptrace.NewTraces()
		span := td.ResourceSpans().AppendEmpty().ScopeSpans().AppendEmpty().Spans().AppendEmpty()
		span.SetTraceID(traceA)
		span.TraceState().FromRaw("congo=t61rcWkgMzE")

		_, shouldProcess, _, err := checkPrerequists(td)
		require.NoError(t, err)
		assert.True(t, shouldProcess)
	})
}
