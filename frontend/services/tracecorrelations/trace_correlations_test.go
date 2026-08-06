package tracecorrelations

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/odigos-io/odigos/frontend/graph/model"
)

func TestPromDuration(t *testing.T) {
	require.Equal(t, "1h", promDuration(time.Hour))
	require.Equal(t, "15m", promDuration(15*time.Minute))
	require.Equal(t, "90s", promDuration(90*time.Second))
	require.Equal(t, "1s", promDuration(500*time.Millisecond))
}

func TestResolveTimeRange(t *testing.T) {
	start := time.Date(2026, 6, 13, 10, 0, 0, 0, time.UTC)
	end := time.Date(2026, 6, 13, 11, 0, 0, 0, time.UTC)

	gotStart, gotEnd, err := resolveTimeRange(&model.TraceCorrelationsTimeRangeInput{
		Start: start.Format(time.RFC3339),
		End:   end.Format(time.RFC3339),
	})
	require.NoError(t, err)
	require.True(t, gotStart.Equal(start))
	require.True(t, gotEnd.Equal(end))
}

func TestResolveTimeRangeInvalid(t *testing.T) {
	_, _, err := resolveTimeRange(&model.TraceCorrelationsTimeRangeInput{
		Start: "not-a-time",
		End:   time.Now().Format(time.RFC3339),
	})
	require.Error(t, err)

	start := time.Now()
	end := start.Add(-time.Hour)
	_, _, err = resolveTimeRange(&model.TraceCorrelationsTimeRangeInput{
		Start: start.Format(time.RFC3339),
		End:   end.Format(time.RFC3339),
	})
	require.Error(t, err)
}
