package clustercollector

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/config"
	"github.com/stretchr/testify/assert"
)

func TestAddSelfTelemetryPipeline(t *testing.T) {
	var empty = struct{}{}
	cases := []struct {
		name string
		cfg  *config.Config
		err  error
	}{
		{
			name: "no pipeline",
			cfg: &config.Config{
				Exporters: map[string]interface{}{},
				Receivers: map[string]interface{}{},
			},
			err: errNoPipelineConfigured,
		},
		{
			name: "no receivers",
			cfg: &config.Config{
				Exporters: map[string]interface{}{},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{},
				},
			},
			err: errNoReceiversConfigured,
		},
		{
			name: "no exporters",
			cfg: &config.Config{
				Receivers: map[string]interface{}{},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{},
				},
			},
			err: errNoExportersConfigured,
		},
		{
			name: "minimal config",
			cfg: &config.Config{
				Receivers: map[string]interface{}{},
				Exporters: map[string]interface{}{},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{},
				},
			},
			err: nil,
		},
		{
			name: "config with pipeline",
			cfg: &config.Config{
				Receivers: config.GenericMap{
					"otlp": config.GenericMap{
						"protocols": config.GenericMap{
							"grpc": empty,
							"http": empty,
						},
					},
				},
				Processors: config.GenericMap{
					"memory_limiter": config.GenericMap{
						"check_interval": "500ms",
					},
					"resource/odigos-version": config.GenericMap{
						"attributes": []config.GenericMap{
							{
								"key":    "odigos.version",
								"value":  "${ODIGOS_VERSION}",
								"action": "upsert",
							},
						},
					},
				},
				Exporters: config.GenericMap{
					"otlp/local": config.GenericMap{
						"endpoint": "http://localhost:4317",
						"tls": config.GenericMap{
							"insecure": true,
						},
					},
				},
				Service: config.Service{
					Pipelines: map[string]config.Pipeline{
						"traces/user-pipeline": {
							Receivers:  []string{"otlp"},
							Processors: []string{"memory_limiter", "resource/odigos-version"},
							Exporters:  []string{"otlp/local"},
						},
					},
				},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.cfg
			err := addSelfTelemetryPipeline(c, k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault, []string{"traces/user-pipeline"}, []string{})
			if !assert.ErrorIs(t, err, tc.err) {
				return
			}
			if err != nil {
				return
			}
			assert.NotEmpty(t, c.Receivers["prometheus/self-metrics"])
			assert.NotEmpty(t, c.Processors["resource/pod-name"])
			assert.NotEmpty(t, c.Processors["resource/odigos-collector-role"])
			assert.NotEmpty(t, c.Service.Pipelines["metrics/otelcol"])
			assert.Equal(t, []string{"prometheus/self-metrics"}, c.Service.Pipelines["metrics/otelcol"].Receivers)
			assert.Equal(t, []string{"resource/pod-name", "resource/odigos-collector-role"}, c.Service.Pipelines["metrics/otelcol"].Processors)
			assert.Equal(t, []string{"otlp/odigos-own-telemetry-ui"}, c.Service.Pipelines["metrics/otelcol"].Exporters)
			pullExporter := c.Service.Telemetry.Metrics.Readers[0]["pull"].(config.GenericMap)["exporter"].(config.GenericMap)
			port := pullExporter["prometheus"].(config.GenericMap)["port"]
			address := pullExporter["prometheus"].(config.GenericMap)["host"]
			assert.Equal(t, k8sconsts.OdigosNodeCollectorOwnTelemetryPortDefault, port)
			assert.Equal(t, "0.0.0.0", address)
			for pipelineName, pipeline := range c.Service.Pipelines {
				if pipelineName == "metrics/otelcol" {
					assert.NotContains(t, pipeline.Processors, "odigostrafficmetrics")
				} else {
					assert.Equal(t, pipeline.Processors[len(pipeline.Processors)-1], "odigostrafficmetrics")
				}

			}
		})
	}
}
