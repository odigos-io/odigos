package common

import (
	"testing"

	odigoscommon "github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMergeProfilingOtlpExporter_Compression(t *testing.T) {
	on := true
	off := false
	assert.Equal(t, "gzip", MergeProfilingOtlpExporter(config.GenericMap{"endpoint": "x", "compression": "none"}, &odigoscommon.OtlpExporterConfiguration{
		EnableDataCompression: &on,
	})["compression"])
	assert.Equal(t, "none", MergeProfilingOtlpExporter(config.GenericMap{"endpoint": "x", "compression": "gzip"}, &odigoscommon.OtlpExporterConfiguration{
		EnableDataCompression: &off,
	})["compression"])
}

func TestMergeProfilingOtlpExporter_NilOtlp(t *testing.T) {
	base := config.GenericMap{"endpoint": "localhost:4317", "compression": "none"}
	out := MergeProfilingOtlpExporter(base, nil)
	assert.Equal(t, base, out)
	out["endpoint"] = "mutated"
	assert.Equal(t, "localhost:4317", base["endpoint"], "mutating result must not change caller base map")
}

func TestMergeProfilingOtlpExporter_WithOtlp_DoesNotMutateBase(t *testing.T) {
	enabled := true
	otlp := &odigoscommon.OtlpExporterConfiguration{
		Timeout: "5s",
		RetryOnFailure: &odigoscommon.RetryOnFailure{
			Enabled: &enabled,
		},
	}
	base := config.GenericMap{"endpoint": "x", "compression": "none"}
	out := MergeProfilingOtlpExporter(base, otlp)
	out["endpoint"] = "y"
	assert.Equal(t, "x", base["endpoint"])
	assert.Equal(t, "5s", out["timeout"])
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

func TestMergeProfilingOtlpExporter_SendingQueue(t *testing.T) {
	enabled := true
	otlp := &odigoscommon.OtlpExporterConfiguration{
		SendingQueue: &odigoscommon.SendingQueue{
			Enabled:   &enabled,
			QueueSize: 50,
		},
	}
	out := MergeProfilingOtlpExporter(config.GenericMap{"endpoint": "x"}, otlp)
	q, ok := out["sending_queue"].(config.GenericMap)
	require.True(t, ok)
	assert.Equal(t, true, q["enabled"])
	assert.Equal(t, 50, q["queue_size"])
}

func TestProfilingProfileDropConditions(t *testing.T) {
	conds := ProfilingProfileDropConditions()
	require.Len(t, conds, 1)
	assert.Equal(t, `resource.attributes["container.id"] == nil`, conds[0])
}

func TestProfilingFilterProcessorConfig(t *testing.T) {
	m := ProfilingFilterProcessorConfig()
	assert.Equal(t, "ignore", m["error_mode"])
	pc, ok := m["profile_conditions"].([]string)
	require.True(t, ok)
	assert.Equal(t, ProfilingProfileDropConditions(), pc)
}
