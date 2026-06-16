package tracecorrelations

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapMetricsStoreError(t *testing.T) {
	require.Nil(t, mapMetricsStoreError(nil))
	require.ErrorIs(t, mapMetricsStoreError(ErrMetricsStoreUnavailable), ErrMetricsStoreUnavailable)
	require.ErrorIs(
		t,
		mapMetricsStoreError(fmt.Errorf("query trace correlation connection counts: %w", errors.New(`Post "http://odigos-correlations-metrics.odigos-system.svc:8428/api/v1/query": dial tcp 10.96.224.201:8428: connect: connection refused`))),
		ErrMetricsStoreUnavailable,
	)
	require.ErrorIs(
		t,
		mapMetricsStoreError(fmt.Errorf("export request failed: %w", errors.New(`Get "http://odigos-correlations-metrics.odigos-system.svc:8428/api/v1/export": dial tcp: lookup odigos-correlations-metrics.odigos-system.svc on 10.96.0.10:53: no such host`))),
		ErrMetricsStoreUnavailable,
	)

	other := errors.New("unexpected query failure")
	require.Equal(t, other, mapMetricsStoreError(other))
}
