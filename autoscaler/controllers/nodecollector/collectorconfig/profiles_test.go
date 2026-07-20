package collectorconfig

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilingPipelineConfig_Disabled(t *testing.T) {
	got := ProfilingPipelineConfig("odigos-system", nil, nil, odigosv1.CollectorsGroupResourcesSettings{})
	assert.Empty(t, got.Receivers)
	assert.Empty(t, got.Processors)
	assert.Empty(t, got.Exporters)
	assert.Empty(t, got.Service.Pipelines)

	off := false
	got = ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &off}, nil, odigosv1.CollectorsGroupResourcesSettings{})
	assert.Empty(t, got.Service.Pipelines)
}

func TestProfilingPipelineConfig_Enabled(t *testing.T) {
	on := true
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &on}, nil, odigosv1.CollectorsGroupResourcesSettings{})
	require.Contains(t, got.Receivers, commonconf.ProfilingReceiver)
	require.Contains(t, got.Processors, commonconf.ProfilingNodeFilterProcessor)
	require.Contains(t, got.Processors, commonconf.ProfilingNodeK8sAttributesProcessor)
	require.Contains(t, got.Processors, commonconf.ProfilingNodeOdigosProfilesProcessor)
	require.Contains(t, got.Processors, commonconf.ProfilingNodeServiceNameProcessor)
	require.Contains(t, got.Exporters, commonconf.ProfilingNodeToGatewayExporter)

	pl, ok := got.Service.Pipelines["profiles"]
	require.True(t, ok)
	assert.Equal(t, []string{commonconf.ProfilingReceiver}, pl.Receivers)
	// Native symbolization is ON by default when profiling is enabled.
	require.Contains(t, got.Processors, commonconf.ProfilingNodeSymbolizeProcessor)
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
		commonconf.ProfilingNodeSymbolizeProcessor,
		commonconf.ProfilingNodeServiceNameProcessor,
	}, pl.Processors)
	assert.Equal(t, []string{commonconf.ProfilingNodeToGatewayExporter}, pl.Exporters)

	filterCfg, ok := got.Processors[commonconf.ProfilingNodeFilterProcessor].(config.GenericMap)
	require.True(t, ok)
	wantFilter := commonconf.ProfilingFilterProcessorConfig()
	assert.Equal(t, wantFilter, filterCfg)

	odigosProfilesCfg, ok := got.Processors[commonconf.ProfilingNodeOdigosProfilesProcessor].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, k8sconsts.OdigosConfigK8sExtensionType, odigosProfilesCfg["odigos_config_extension"])
}

func TestProfilingPipelineConfig_UserProcessorsAppended(t *testing.T) {
	on := true
	userProcessors := []string{"resource/addclusterinfo", "transform/rename"}
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &on}, userProcessors, odigosv1.CollectorsGroupResourcesSettings{})

	pl, ok := got.Service.Pipelines["profiles"]
	require.True(t, ok)
	// User processors run after the built-in enrichment chain (native symbolization is on by
	// default, so the symbolize processor is present) and before export.
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
		commonconf.ProfilingNodeSymbolizeProcessor,
		commonconf.ProfilingNodeServiceNameProcessor,
		"resource/addclusterinfo",
		"transform/rename",
	}, pl.Processors)
}

// TestProfilingPipelineConfig_NativeSymbolizationDisabled drops the symbolize
// processor when a user explicitly opts out (profiling.symbolization.native: false).
func TestProfilingPipelineConfig_NativeSymbolizationDisabled(t *testing.T) {
	on, off := true, false
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{
		Enabled:       &on,
		Symbolization: &common.ProfilingSymbolizationConfiguration{Native: &off},
	}, nil, odigosv1.CollectorsGroupResourcesSettings{})
	require.NotContains(t, got.Processors, commonconf.ProfilingNodeSymbolizeProcessor)

	pl := got.Service.Pipelines["profiles"]
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
		commonconf.ProfilingNodeServiceNameProcessor,
	}, pl.Processors)
}
