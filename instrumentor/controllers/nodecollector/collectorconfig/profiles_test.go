package collectorconfig

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilingPipelineConfig_Disabled(t *testing.T) {
	got := ProfilingPipelineConfig("odigos-system", nil, nil)
	assert.Empty(t, got.Receivers)
	assert.Empty(t, got.Processors)
	assert.Empty(t, got.Exporters)
	assert.Empty(t, got.Service.Pipelines)

	off := false
	got = ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &off}, nil)
	assert.Empty(t, got.Service.Pipelines)
}

func TestProfilingPipelineConfig_Enabled(t *testing.T) {
	on := true
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &on}, nil)
	require.Contains(t, got.Receivers, pipeline.ProfilingReceiver)
	require.Contains(t, got.Processors, pipeline.ProfilingNodeFilterProcessor)
	require.Contains(t, got.Processors, pipeline.ProfilingNodeK8sAttributesProcessor)
	require.Contains(t, got.Processors, pipeline.ProfilingNodeOdigosProfilesProcessor)
	require.Contains(t, got.Processors, pipeline.ProfilingNodeServiceNameProcessor)
	require.Contains(t, got.Exporters, pipeline.ProfilingNodeToGatewayExporter)

	pl, ok := got.Service.Pipelines["profiles"]
	require.True(t, ok)
	assert.Equal(t, []string{pipeline.ProfilingReceiver}, pl.Receivers)
	// Native symbolization is ON by default when profiling is enabled.
	require.Contains(t, got.Processors, pipeline.ProfilingNodeSymbolizeProcessor)
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		pipeline.ProfilingNodeFilterProcessor,
		pipeline.ProfilingNodeK8sAttributesProcessor,
		pipeline.ProfilingNodeOdigosProfilesProcessor,
		pipeline.ProfilingNodeSymbolizeProcessor,
		pipeline.ProfilingNodeServiceNameProcessor,
		odigosTrafficMetricsProcessorName,
	}, pl.Processors)
	assert.Equal(t, []string{pipeline.ProfilingNodeToGatewayExporter}, pl.Exporters)

	filterCfg, ok := got.Processors[pipeline.ProfilingNodeFilterProcessor].(config.GenericMap)
	require.True(t, ok)
	wantFilter := pipeline.ProfilingFilterProcessorConfig()
	assert.Equal(t, wantFilter, filterCfg)

	odigosProfilesCfg, ok := got.Processors[pipeline.ProfilingNodeOdigosProfilesProcessor].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, k8sconsts.OdigosConfigK8sExtensionType, odigosProfilesCfg["odigos_config_extension"])
}

func TestProfilingPipelineConfig_UserProcessorsAppended(t *testing.T) {
	on := true
	userProcessors := []string{"resource/addclusterinfo", "transform/rename"}
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &on}, userProcessors)

	pl, ok := got.Service.Pipelines["profiles"]
	require.True(t, ok)
	// User processors run after the built-in enrichment chain (native symbolization is on by
	// default, so the symbolize processor is present) and before export.
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		pipeline.ProfilingNodeFilterProcessor,
		pipeline.ProfilingNodeK8sAttributesProcessor,
		pipeline.ProfilingNodeOdigosProfilesProcessor,
		pipeline.ProfilingNodeSymbolizeProcessor,
		pipeline.ProfilingNodeServiceNameProcessor,
		"resource/addclusterinfo",
		"transform/rename",
		odigosTrafficMetricsProcessorName,
	}, pl.Processors)
}

// TestProfilingPipelineConfig_NativeSymbolizationDisabled drops the symbolize
// processor when a user explicitly opts out (profiling.symbolization.native: false).
func TestProfilingPipelineConfig_NativeSymbolizationDisabled(t *testing.T) {
	on, off := true, false
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{
		Enabled:       &on,
		Symbolization: &common.ProfilingSymbolizationConfiguration{Native: &off},
	}, nil)
	require.NotContains(t, got.Processors, pipeline.ProfilingNodeSymbolizeProcessor)

	pl := got.Service.Pipelines["profiles"]
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		pipeline.ProfilingNodeFilterProcessor,
		pipeline.ProfilingNodeK8sAttributesProcessor,
		pipeline.ProfilingNodeOdigosProfilesProcessor,
		pipeline.ProfilingNodeServiceNameProcessor,
		odigosTrafficMetricsProcessorName,
	}, pl.Processors)
}
