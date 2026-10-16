package config

import (
	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
	"testing"
)

func TestGoogleManagedPrometheusConfig(t *testing.T) {
	for _, data := range []map[string]string{nil, {"GCP_PROJECT_ID": "project", "GCP_TIMEOUT": "20s", "GMP_LOCATION": "us-central1", "GMP_CLUSTER": "cluster"}} {
		configers, err := LoadConfigers()
		require.NoError(t, err)
		g := configers[common.GoogleManagedPrometheusDestinationType]
		require.NotNil(t, g)
		cfg := Config{Exporters: GenericMap{}, Processors: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		pipelines, err := g.ModifyConfig(&mockDestination{id: "test", config: data}, &cfg)
		require.NoError(t, err)
		require.Equal(t, []string{"metrics/googlemanagedprometheus-test"}, pipelines)
		require.Len(t, cfg.Service.Pipelines, 1)
		exporter := cfg.Exporters["googlemanagedprometheus/test"].(GenericMap)
		require.Equal(t, "gzip", exporter["metric"].(GenericMap)["compression"])
		require.NotContains(t, exporter, "retry_on_failure")
		if data == nil {
			require.NotContains(t, exporter, "project")
			require.Len(t, cfg.Service.Pipelines[pipelines[0]].Processors, 1)
		} else {
			require.Equal(t, "project", exporter["project"])
			require.Equal(t, "20s", exporter["timeout"])
			require.Equal(t, []string{"resource/googlemanagedprometheus-test", "transform/googlemanagedprometheus-test"}, cfg.Service.Pipelines[pipelines[0]].Processors)
		}
	}
}

func TestGoogleManagedPrometheusRejectsUnsupportedConfig(t *testing.T) {
	for _, dest := range []*mockDestination{
		{signals: []common.ObservabilitySignal{common.TracesObservabilitySignal, common.LogsObservabilitySignal, common.ProfilesObservabilitySignal}},
		{config: map[string]string{"GCP_TIMEOUT": "bad"}},
		{config: map[string]string{"GMP_LOCATION": "global"}},
		{config: map[string]string{"GCP_TIMEOUT": "0s"}},
		{config: map[string]string{"GCP_TIMEOUT": "-1s"}},
	} {
		cfg := Config{Exporters: GenericMap{}, Processors: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
		_, err := (&GoogleManagedPrometheus{}).ModifyConfig(dest, &cfg)
		require.Error(t, err)
		require.Empty(t, cfg.Exporters)
		require.Empty(t, cfg.Processors)
		require.Empty(t, cfg.Service.Pipelines)
	}
}
