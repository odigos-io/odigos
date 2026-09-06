package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRemoteWriteEndpoint(t *testing.T) {
	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "host only appends default write path",
			url:  "http://prometheus:9090",
			want: "http://prometheus:9090/api/v1/write",
		},
		{
			name: "existing write suffix kept as-is",
			url:  "http://prometheus:9090/api/v1/write",
			want: "http://prometheus:9090/api/v1/write",
		},
		{
			name: "mimir push suffix kept as-is",
			url:  "http://mimir-distributed-gateway.mimir/api/v1/push",
			want: "http://mimir-distributed-gateway.mimir/api/v1/push",
		},
		{
			name: "unknown path appends default write path",
			url:  "http://prometheus:9090/custom",
			want: "http://prometheus:9090/custom/api/v1/write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, remoteWriteEndpoint(tt.url))
		})
	}
}

func TestPrometheusModifyConfigEndpoint(t *testing.T) {
	tests := []struct {
		name         string
		url          string
		wantEndpoint string
	}{
		{
			name:         "default write path appended for mimir push",
			url:          "http://mimir-distributed-gateway.mimir/api/v1/push",
			wantEndpoint: "http://mimir-distributed-gateway.mimir/api/v1/push",
		},
		{
			name:         "host only appends default write path",
			url:          "http://prometheus:9090",
			wantEndpoint: "http://prometheus:9090/api/v1/write",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := &mockDestination{
				id: "test-id",
				config: map[string]string{
					promRWurlKey: tt.url,
				},
			}

			config := &Config{
				Extensions: make(map[string]interface{}),
				Exporters:  make(map[string]interface{}),
				Service: Service{
					Extensions: []string{},
					Pipelines:  make(map[string]Pipeline),
				},
			}

			p := &Prometheus{}
			_, err := p.ModifyConfig(dest, config)
			assert.NoError(t, err)

			exporter := config.Exporters["prometheusremotewrite/prometheus-test-id"].(GenericMap)
			assert.Equal(t, tt.wantEndpoint, exporter["endpoint"])
		})
	}
}
