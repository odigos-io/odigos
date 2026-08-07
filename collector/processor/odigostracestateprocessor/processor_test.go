package odigostracestateprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/odigosattributes"
)

func TestProcessSpanTraceStateDryRun(t *testing.T) {
	tests := []struct {
		name         string
		traceState   string
		expectedKept *bool
	}{
		{
			name:         "kept",
			traceState:   "odigos=dry:t",
			expectedKept: boolPtr(true),
		},
		{
			name:         "dropped",
			traceState:   "odigos=dry:f",
			expectedKept: boolPtr(false),
		},
		{
			name:         "unknown decision",
			traceState:   "odigos=dry:x",
			expectedKept: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			span := ptrace.NewSpan()
			span.TraceState().FromRaw(tt.traceState)

			processor := traceStateProcessor{
				logger: zap.NewNop(),
			}
			processor.processSpanTraceState(span)

			dryRun, ok := span.Attributes().Get(odigosattributes.SamplingDryRun)
			require.True(t, ok)
			require.True(t, dryRun.Bool())

			traceKept, ok := span.Attributes().Get(odigosattributes.SamplingTraceKept)
			if tt.expectedKept == nil {
				require.False(t, ok)
				return
			}
			require.True(t, ok)
			require.Equal(t, *tt.expectedKept, traceKept.Bool())
		})
	}
}

func boolPtr(value bool) *bool {
	return &value
}
