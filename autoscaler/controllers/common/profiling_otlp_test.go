package common

import (
	"testing"

	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	odigosconsts "github.com/odigos-io/odigos/common/consts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeProfilingOtlpExporter_NilOtlp(t *testing.T) {
	base := config.GenericMap{"endpoint": "localhost:4317", "compression": "none"}
	out := MergeProfilingOtlpExporter(base, nil)
	assert.Equal(t, base, out)
}

func TestMergeProfilingOtlpExporter_TimeoutAndRetry(t *testing.T) {
	enabled := true
	otlp := &odigoscommon.OtlpExporterConfiguration{
		Timeout: "5s",
		RetryOnFailure: &odigoscommon.RetryOnFailure{
			Enabled:         &enabled,
			InitialInterval: "1s",
			MaxInterval:     "30s",
		},
	}
	out := MergeProfilingOtlpExporter(config.GenericMap{"endpoint": "x"}, otlp)
	assert.Equal(t, "5s", out["timeout"])
	retry, ok := out["retry_on_failure"].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, true, retry["enabled"])
	assert.Equal(t, "1s", retry["initial_interval"])
	assert.Equal(t, "30s", retry["max_interval"])
}

func TestProfilingProfileDropConditions(t *testing.T) {
	conds := ProfilingProfileDropConditions()
	require.Len(t, conds, 2)
	assert.Equal(t, `resource.attributes["container.id"] == nil`, conds[0])
	assert.Contains(t, conds[1], odigosconsts.OdigosCollectorTelemetryServiceName)
	assert.Contains(t, conds[1], `resource.attributes["service.name"]`)
}

func TestProfilingFilterProcessorConfig(t *testing.T) {
	m := ProfilingFilterProcessorConfig()
	assert.Equal(t, "ignore", m["error_mode"])
	pc, ok := m["profile_conditions"].([]string)
	require.True(t, ok)
	assert.Equal(t, ProfilingProfileDropConditions(), pc)
}
