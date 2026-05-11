package odigostracefilterprocessor

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

func TestDropUnsampledSpans(t *testing.T) {
	tests := []struct {
		name          string
		flags         uint32
		dropUnsampled bool
		expectDropped bool
	}{
		{
			name:          "flags=0, drop_unsampled=true -> dropped",
			flags:         0,
			dropUnsampled: true,
			expectDropped: true,
		},
		{
			name:          "flags=1, drop_unsampled=true -> kept (sampled bit set)",
			flags:         1,
			dropUnsampled: true,
			expectDropped: false,
		},
		{
			name:          "flags=3, drop_unsampled=true -> kept (sampled bit set, other bits too)",
			flags:         3,
			dropUnsampled: true,
			expectDropped: false,
		},
		{
			name:          "flags=2, drop_unsampled=true -> dropped (bit 1 set, but sampled bit 0 not set)",
			flags:         2,
			dropUnsampled: true,
			expectDropped: true,
		},
		{
			name:          "flags=0, drop_unsampled=false -> no filtering",
			flags:         0,
			dropUnsampled: false,
			expectDropped: false,
		},
		{
			name:          "flags=5 (0b101), drop_unsampled=true -> kept (sampled bit set)",
			flags:         5,
			dropUnsampled: true,
			expectDropped: false,
		},
		{
			name:          "flags=4 (0b100), drop_unsampled=true -> dropped (sampled bit not set)",
			flags:         4,
			dropUnsampled: true,
			expectDropped: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var evaluators []SpanFilterEvaluator
			if tt.dropUnsampled {
				evaluators = append(evaluators, &unsampledBitEvaluator{})
			}

			proc := &traceFilterProcessor{
				logger:     zap.NewNop(),
				evaluators: evaluators,
			}

			td := createTestTraces(tt.flags)
			result, err := proc.processTraces(context.Background(), td)
			require.NoError(t, err)

			if tt.expectDropped {
				assert.Equal(t, 0, countSpans(result))
			} else {
				assert.Equal(t, 1, countSpans(result))
			}
		})
	}
}

func TestNoEvaluatorsPassesThrough(t *testing.T) {
	proc := &traceFilterProcessor{
		logger:     zap.NewNop(),
		evaluators: nil,
	}

	td := createTestTraces(0)
	result, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Equal(t, 1, countSpans(result))
}

func TestMultipleSpansMixedFlags(t *testing.T) {
	proc := &traceFilterProcessor{
		logger:     zap.NewNop(),
		evaluators: []SpanFilterEvaluator{&unsampledBitEvaluator{}},
	}

	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	ss := rs.ScopeSpans().AppendEmpty()

	span1 := ss.Spans().AppendEmpty()
	span1.SetName("sampled")
	span1.SetFlags(1)

	span2 := ss.Spans().AppendEmpty()
	span2.SetName("unsampled")
	span2.SetFlags(0)

	span3 := ss.Spans().AppendEmpty()
	span3.SetName("sampled-multi-flag")
	span3.SetFlags(3)

	result, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Equal(t, 2, countSpans(result))

	spans := result.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
	assert.Equal(t, "sampled", spans.At(0).Name())
	assert.Equal(t, "sampled-multi-flag", spans.At(1).Name())
}

func TestRubyResourceKeepsSpansWithoutSampledBit(t *testing.T) {
	proc := &traceFilterProcessor{
		logger:     zap.NewNop(),
		evaluators: []SpanFilterEvaluator{&unsampledBitEvaluator{}},
	}

	td := ptrace.NewTraces()

	rubyResource := td.ResourceSpans().AppendEmpty()
	rubyResource.Resource().Attributes().PutStr(telemetrySDKLanguageAttributeName, rubyTelemetrySDKLanguage)
	rubySpan := rubyResource.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	rubySpan.SetName("ruby-span")
	rubySpan.SetFlags(0)

	pythonResource := td.ResourceSpans().AppendEmpty()
	pythonResource.Resource().Attributes().PutStr(telemetrySDKLanguageAttributeName, "python")
	pythonSpan := pythonResource.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	pythonSpan.SetName("python-unsampled")
	pythonSpan.SetFlags(0)

	result, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Equal(t, 1, countSpans(result))
	require.Equal(t, 1, result.ResourceSpans().Len())

	spans := result.ResourceSpans().At(0).ScopeSpans().At(0).Spans()
	require.Equal(t, 1, spans.Len())
	assert.Equal(t, "ruby-span", spans.At(0).Name())
}

func TestEmptyResourceSpansRemoved(t *testing.T) {
	proc := &traceFilterProcessor{
		logger:     zap.NewNop(),
		evaluators: []SpanFilterEvaluator{&unsampledBitEvaluator{}},
	}

	td := ptrace.NewTraces()
	rs1 := td.ResourceSpans().AppendEmpty()
	rs1.Resource().Attributes().PutStr("service.name", "svc1")
	span1 := rs1.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span1.SetFlags(0)

	rs2 := td.ResourceSpans().AppendEmpty()
	rs2.Resource().Attributes().PutStr("service.name", "svc2")
	span2 := rs2.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span2.SetFlags(1)

	result, err := proc.processTraces(context.Background(), td)
	require.NoError(t, err)
	assert.Equal(t, 1, result.ResourceSpans().Len())

	svcName, ok := result.ResourceSpans().At(0).Resource().Attributes().Get("service.name")
	require.True(t, ok)
	assert.Equal(t, "svc2", svcName.Str())
}

func createTestTraces(flags uint32) ptrace.Traces {
	td := ptrace.NewTraces()
	rs := td.ResourceSpans().AppendEmpty()
	rs.Resource().Attributes().PutStr("resource", "R1")
	span := rs.ScopeSpans().AppendEmpty().Spans().AppendEmpty()
	span.SetName("test_span")
	span.SetFlags(flags)
	return td
}

func countSpans(td ptrace.Traces) int {
	count := 0
	for i := 0; i < td.ResourceSpans().Len(); i++ {
		rs := td.ResourceSpans().At(i)
		for j := 0; j < rs.ScopeSpans().Len(); j++ {
			count += rs.ScopeSpans().At(j).Spans().Len()
		}
	}
	return count
}
