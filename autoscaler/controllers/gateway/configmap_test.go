package gateway

import (
	"fmt"
	"testing"

	"github.com/odigos-io/odigos/common/config"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
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
						"check_interval": "1s",
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
			err := addSelfTelemetryPipeline(c)
			if !assert.ErrorIs(t, err, tc.err) {
				return
			}
			if err != nil {
				return
			}
			assert.NotEmpty(t, c.Receivers["prometheus"])
			assert.NotEmpty(t, c.Processors["resource/pod-name"])
			assert.NotEmpty(t, c.Service.Pipelines["metrics/otelcol"])
			assert.Equal(t, []string{"prometheus"}, c.Service.Pipelines["metrics/otelcol"].Receivers)
			assert.Equal(t, []string{"resource/pod-name"}, c.Service.Pipelines["metrics/otelcol"].Processors)
			assert.Equal(t, []string{"otlp/ui"}, c.Service.Pipelines["metrics/otelcol"].Exporters)
			assert.Equal(t, fmt.Sprintf("0.0.0.0:%d", consts.OdigosNodeCollectorOwnTelemetryPortDefault), c.Service.Telemetry.Metrics["address"])
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
