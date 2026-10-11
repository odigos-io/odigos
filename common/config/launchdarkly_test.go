package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

func TestLaunchDarklyModifyConfig(t *testing.T) {
	for _, signals := range [][]common.ObservabilitySignal{
		{common.TracesObservabilitySignal},
		{common.MetricsObservabilitySignal},
		{common.LogsObservabilitySignal},
		{common.TracesObservabilitySignal, common.MetricsObservabilitySignal, common.LogsObservabilitySignal},
	} {
		dest := &mockDestination{id: "odigos.io.dest.launchdarkly-test", signals: signals}
		cfg := &Config{
			Exporters:  GenericMap{},
			Processors: GenericMap{},
			Service:    Service{Pipelines: map[string]Pipeline{}},
		}
		pipelines, err := (&LaunchDarkly{}).ModifyConfig(dest, cfg)
		require.NoError(t, err)
		require.Len(t, pipelines, len(signals))

		exporterName := "otlphttp/launchdarkly-" + dest.GetID()
		exporter := cfg.Exporters[exporterName].(GenericMap)
		require.Equal(t, "https://otel.observability.app.launchdarkly.com", exporter["endpoint"])
		require.Equal(t, "gzip", exporter["compression"])
		require.NotContains(t, exporter, "headers")
		batch := exporter["sending_queue"].(GenericMap)["batch"].(GenericMap)
		require.Equal(t, "items", batch["sizer"])
		require.Equal(t, 512, batch["max_size"])
		require.LessOrEqual(t, batch["min_size"].(int), batch["max_size"].(int))

		processorName := "resource/launchdarkly-" + dest.GetID()
		attributes := cfg.Processors[processorName].(GenericMap)["attributes"].([]GenericMap)
		require.Equal(t, []GenericMap{{
			"key":    "launchdarkly.project_id",
			"value":  "${ODIGOS_DEST_odigos_io_dest_launchdarkly_test_LAUNCHDARKLY_SDK_KEY}",
			"action": "upsert",
		}}, attributes)
		for _, pipelineName := range pipelines {
			pipeline := cfg.Service.Pipelines[pipelineName]
			require.Equal(t, []string{exporterName}, pipeline.Exporters)
			require.Equal(t, []string{processorName}, pipeline.Processors)
		}
	}
}

func TestLaunchDarklyDestinationSecretsAreScoped(t *testing.T) {
	for _, id := range []string{"first", "second"} {
		dest := &mockDestination{id: id}
		cfg := &Config{
			Exporters: GenericMap{}, Processors: GenericMap{},
			Service: Service{Pipelines: map[string]Pipeline{}},
		}
		_, err := (&LaunchDarkly{}).ModifyConfig(dest, cfg)
		require.NoError(t, err)
		attributes := cfg.Processors["resource/launchdarkly-"+id].(GenericMap)["attributes"].([]GenericMap)
		require.Equal(t, "${ODIGOS_DEST_"+id+"_LAUNCHDARKLY_SDK_KEY}", attributes[0]["value"])
	}
}
