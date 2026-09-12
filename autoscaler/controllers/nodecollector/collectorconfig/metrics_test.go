package collectorconfig

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsConfig_MetricsPipelineProcessorOrder(t *testing.T) {
	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
		MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{
			AgentsTelemetry: &odigosv1.AgentsTelemetrySettings{},
		},
	})

	pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
	require.True(t, ok)
	// traffic accounting must stay last so counts reflect the final metric shape.
	require.NotEmpty(t, pl.Processors)
	assert.Equal(t, odigosTrafficMetricsProcessorName, pl.Processors[len(pl.Processors)-1])
}

// JVM runtime metrics from the eBPF Java agent arrive on otlp/in and are
// reported regardless of agentsTelemetry, so switching that off must not take
// them away - and must not let any other OTLP metric through.
func TestMetricsConfig_JVMRuntimeMetricsPipeline(t *testing.T) {
	for _, tc := range []struct {
		name            string
		tier            common.OdigosTier
		agentsTelemetry *odigosv1.AgentsTelemetrySettings
		wantPipeline    bool
	}{
		{"enterprise without agents telemetry keeps JVM runtime metrics", common.OnPremOdigosTier, nil, true},
		{"enterprise with agents telemetry already routes otlp/in", common.OnPremOdigosTier, &odigosv1.AgentsTelemetrySettings{}, false},
		{"community has no eBPF Java agent", common.CommunityOdigosTier, nil, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
				CommonSignalConfig:    CommonSignalConfig{Tier: tc.tier},
				MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{AgentsTelemetry: tc.agentsTelemetry},
			})

			pl, ok := cfg.Service.Pipelines[jvmRuntimeMetricsPipelineName]
			require.Equal(t, tc.wantPipeline, ok)
			if !tc.wantPipeline {
				return
			}
			assert.Equal(t, []string{OTLPInReceiverName}, pl.Receivers)
			assert.Contains(t, pl.Processors, jvmRuntimeMetricsFilterName)
			assert.Equal(t, odigosTrafficMetricsProcessorName, pl.Processors[len(pl.Processors)-1])
			assert.Equal(t, []string{clusterCollectorMetricsExporterName}, pl.Exporters)

			filter, ok := cfg.Processors[jvmRuntimeMetricsFilterName].(config.GenericMap)
			require.True(t, ok, "filter processor must be defined alongside the pipeline")
			conditions := filter["metrics"].(config.GenericMap)["metric"].([]string)
			assert.Equal(t, []string{`instrumentation_scope.name != "jvm-ebpf-metrics"`}, conditions)

			// The main pipeline must not pick up otlp/in, or JVM metrics would be exported twice.
			assert.NotContains(t, cfg.Service.Pipelines[odigosMetricsPipelineName].Receivers, OTLPInReceiverName)
		})
	}
}
