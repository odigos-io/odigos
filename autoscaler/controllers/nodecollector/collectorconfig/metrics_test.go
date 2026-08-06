package collectorconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsConfig_MetricsPipelineProcessorOrder(t *testing.T) {
	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
		MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{},
	})

	pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
	require.True(t, ok)
	// traffic accounting must stay last so counts reflect the final metric shape.
	require.NotEmpty(t, pl.Processors)
	assert.Equal(t, odigosTrafficMetricsProcessorName, pl.Processors[len(pl.Processors)-1])
}
