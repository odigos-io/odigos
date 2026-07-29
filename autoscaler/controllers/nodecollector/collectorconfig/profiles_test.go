package collectorconfig

import (
	"testing"

	"github.com/go-logr/logr"
	"github.com/go-logr/logr/funcr"
	"github.com/odigos-io/odigos/api/k8sconsts"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfilingPipelineConfig_Disabled(t *testing.T) {
	got := ProfilingPipelineConfig(logr.Discard(), "odigos-system", nil, nil)
	assert.Empty(t, got.Receivers)
	assert.Empty(t, got.Processors)
	assert.Empty(t, got.Exporters)
	assert.Empty(t, got.Service.Pipelines)

	off := false
	got = ProfilingPipelineConfig(logr.Discard(), "odigos-system", &common.ProfilingConfiguration{Enabled: &off}, nil)
	assert.Empty(t, got.Service.Pipelines)
}

func TestProfilingPipelineConfig_Enabled(t *testing.T) {
	on := true
	got := ProfilingPipelineConfig(logr.Discard(), "odigos-system", &common.ProfilingConfiguration{Enabled: &on}, nil)
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
		odigosTrafficMetricsProcessorName,
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
	got := ProfilingPipelineConfig(logr.Discard(), "odigos-system", &common.ProfilingConfiguration{Enabled: &on}, userProcessors)

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
		odigosTrafficMetricsProcessorName,
	}, pl.Processors)
}

// TestProfilingPipelineConfig_UserSymbolizeProcessorWarns covers the escape hatch a
// user can reach via the Processor CRD: a manifest processor of type
// odigossymbolizeprocessor runs as its own instance, independent of maxMemoryMiB.
// That's allowed (it's the same general capability every CRD-managed processor
// has), but it used to be silent -- this asserts it's now surfaced via a Warn log.
func TestProfilingPipelineConfig_UserSymbolizeProcessorWarns(t *testing.T) {
	on := true

	var logs []string
	logger := funcr.New(func(_, args string) { logs = append(logs, args) }, funcr.Options{})

	got := ProfilingPipelineConfig(logger, "odigos-system",
		&common.ProfilingConfiguration{Enabled: &on},
		[]string{"odigossymbolizeprocessor/my-custom-instance"})

	require.Contains(t, got.Service.Pipelines["profiles"].Processors, "odigossymbolizeprocessor/my-custom-instance")
	require.Len(t, logs, 1)
	assert.Contains(t, logs[0], "odigossymbolizeprocessor/my-custom-instance")

	// The built-in UserProcessorsAppended case (no odigossymbolizeprocessor-typed
	// manifest processor) must not warn.
	logs = nil
	_ = ProfilingPipelineConfig(logger, "odigos-system",
		&common.ProfilingConfiguration{Enabled: &on},
		[]string{"resource/addclusterinfo", "transform/rename"})
	assert.Empty(t, logs)
}

// TestProfilingPipelineConfig_NativeSymbolizationDisabled drops the symbolize
// processor when a user explicitly opts out (profiling.symbolization.native: false).
func TestProfilingPipelineConfig_NativeSymbolizationDisabled(t *testing.T) {
	on, off := true, false
	got := ProfilingPipelineConfig(logr.Discard(), "odigos-system", &common.ProfilingConfiguration{
		Enabled:       &on,
		Symbolization: &common.ProfilingSymbolizationConfiguration{Native: &off},
	}, nil)
	require.NotContains(t, got.Processors, commonconf.ProfilingNodeSymbolizeProcessor)

	pl := got.Service.Pipelines["profiles"]
	assert.Equal(t, []string{
		memoryLimiterProcessorName,
		commonconf.ProfilingNodeFilterProcessor,
		commonconf.ProfilingNodeK8sAttributesProcessor,
		commonconf.ProfilingNodeOdigosProfilesProcessor,
		commonconf.ProfilingNodeServiceNameProcessor,
		odigosTrafficMetricsProcessorName,
	}, pl.Processors)
}
