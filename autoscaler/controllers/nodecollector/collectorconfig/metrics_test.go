package collectorconfig

import (
	"slices"
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/agentsignalconfig"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetricsConfig_ObiRenameProcessorOmittedWhenNoNetworkMetrics(t *testing.T) {
	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
		MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{},
		NetworkMetricsEnabled: false,
	})

	assert.NotContains(t, cfg.Processors, obiMetricsRenameProcessorName)
	pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
	require.True(t, ok)
	assert.NotContains(t, pl.Processors, obiMetricsRenameProcessorName)
}

func TestMetricsConfig_ObiRenameProcessorAddedWhenNetworkMetrics(t *testing.T) {
	cfg := MetricsConfig(&odigosv1.CollectorsGroup{}, MetricsConfigOptions{
		MetricsConfigSettings: &odigosv1.CollectorsGroupMetricsCollectionSettings{},
		NetworkMetricsEnabled: true,
	})

	require.Contains(t, cfg.Processors, obiMetricsRenameProcessorName)
	pl, ok := cfg.Service.Pipelines[odigosMetricsPipelineName]
	require.True(t, ok)
	require.Contains(t, pl.Processors, obiMetricsRenameProcessorName)

	// the rename must run before traffic accounting so counts reflect the final metric names.
	renameIdx := slices.Index(pl.Processors, obiMetricsRenameProcessorName)
	trafficIdx := slices.Index(pl.Processors, odigosTrafficMetricsProcessorName)
	assert.Less(t, renameIdx, trafficIdx)
}

func TestAnyNetworkMetricsEnabled(t *testing.T) {
	assert.False(t, AnyNetworkMetricsEnabled(nil))
	assert.False(t, AnyNetworkMetricsEnabled(&odigosv1.InstrumentationConfigList{}))

	withoutNetworkMetrics := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			{Spec: odigosv1.InstrumentationConfigSpec{
				Containers: []odigosv1.ContainerAgentConfig{
					{Metrics: &agentsignalconfig.AgentMetricsConfig{}},
				},
			}},
		},
	}
	assert.False(t, AnyNetworkMetricsEnabled(withoutNetworkMetrics))

	withNetworkMetrics := &odigosv1.InstrumentationConfigList{
		Items: []odigosv1.InstrumentationConfig{
			{Spec: odigosv1.InstrumentationConfigSpec{
				Containers: []odigosv1.ContainerAgentConfig{
					{Metrics: &agentsignalconfig.AgentMetricsConfig{}},
					{Metrics: &agentsignalconfig.AgentMetricsConfig{
						NetworkMetrics: &instrumentationrules.NetworkMetricsConfig{},
					}},
				},
			}},
		},
	}
	assert.True(t, AnyNetworkMetricsEnabled(withNetworkMetrics))
}
