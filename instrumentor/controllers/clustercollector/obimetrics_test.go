package clustercollector

import (
	"testing"

	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	pipelinegen "github.com/odigos-io/odigos/common/pipelinegen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAddObiMetricsRenamePipeline_MetricsEnabled(t *testing.T) {
	metricsRoot := pipelinegen.GetTelemetryRootPipelineName(odigoscommon.MetricsObservabilitySignal)
	c := &config.Config{
		Processors: config.GenericMap{},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				metricsRoot: {
					Receivers:  []string{"otlp"},
					Processors: []string{resourceOdigosVersionProcessorName, "batch"},
					Exporters:  []string{"odigosrouterconnector/metrics"},
				},
			},
		},
	}

	addObiMetricsRenamePipeline(c)

	require.Contains(t, c.Processors, obiMetricsRenameProcessorName)
	// rename runs right after resource/odigos-version and before user processors.
	assert.Equal(t,
		[]string{resourceOdigosVersionProcessorName, obiMetricsRenameProcessorName, "batch"},
		c.Service.Pipelines[metricsRoot].Processors,
	)
}

func TestAddObiMetricsRenamePipeline_MetricsDisabled(t *testing.T) {
	c := &config.Config{
		Processors: config.GenericMap{},
		Service: config.Service{
			Pipelines: map[string]config.Pipeline{
				pipelinegen.GetTelemetryRootPipelineName(odigoscommon.TracesObservabilitySignal): {
					Receivers:  []string{"otlp"},
					Processors: []string{resourceOdigosVersionProcessorName},
					Exporters:  []string{"odigosrouterconnector/traces"},
				},
			},
		},
	}

	addObiMetricsRenamePipeline(c)

	assert.NotContains(t, c.Processors, obiMetricsRenameProcessorName)
}
