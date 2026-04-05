package collectorconfig

import (
	"testing"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilingPipelineConfig_Disabled(t *testing.T) {
	got := ProfilingPipelineConfig("odigos-system", nil)
	assert.Empty(t, got.Receivers)
	assert.Empty(t, got.Processors)
	assert.Empty(t, got.Exporters)
	assert.Empty(t, got.Service.Pipelines)

	off := false
	got = ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &off})
	assert.Empty(t, got.Service.Pipelines)
}

func TestProfilingPipelineConfig_Enabled(t *testing.T) {
	on := true
	got := ProfilingPipelineConfig("odigos-system", &common.ProfilingConfiguration{Enabled: &on})
	require.Contains(t, got.Receivers, commonconf.ProfilingReceiver)
	require.Contains(t, got.Processors, commonconf.ProfilingNodeFilterProcessor)
	require.Contains(t, got.Processors, commonconf.ProfilingNodeK8sAttributesProcessor)
	require.Contains(t, got.Exporters, commonconf.ProfilingNodeToGatewayExporter)

	pl, ok := got.Service.Pipelines["profiles"]
	require.True(t, ok)
	assert.Equal(t, []string{commonconf.ProfilingReceiver}, pl.Receivers)
	assert.Equal(t, []string{commonconf.ProfilingNodeFilterProcessor, commonconf.ProfilingNodeK8sAttributesProcessor}, pl.Processors)
	assert.Equal(t, []string{commonconf.ProfilingNodeToGatewayExporter}, pl.Exporters)

	filterCfg, ok := got.Processors[commonconf.ProfilingNodeFilterProcessor].(config.GenericMap)
	require.True(t, ok)
	wantFilter := commonconf.ProfilingFilterProcessorConfig()
	assert.Equal(t, wantFilter, filterCfg)
}
