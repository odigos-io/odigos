package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogzioNormalizeRegion(t *testing.T) {
	tests := []struct {
		name   string
		region string
		want   string
	}{
		// every value the LOGZIO_REGION dropdown in destinations/data/logzio.yaml offers
		{name: "dropdown us", region: "listener.logz.io", want: "us"},
		{name: "dropdown au", region: "listener-au.logz.io", want: "au"},
		{name: "dropdown ca", region: "listener-ca.logz.io", want: "ca"},
		{name: "dropdown eu", region: "listener-eu.logz.io", want: "eu"},
		{name: "dropdown nl", region: "listener-nl.logz.io", want: "nl"},
		{name: "dropdown uk", region: "listener-uk.logz.io", want: "uk"},
		{name: "dropdown wa", region: "listener-wa.logz.io", want: "wa"},
		// a hand written Destination may carry the two letter code the docs describe
		{name: "legacy two letter code", region: "eu", want: "eu"},
		{name: "upper case hostname", region: "Listener-EU.logz.io", want: "eu"},
		{name: "surrounding whitespace", region: " listener-uk.logz.io ", want: "uk"},
		{name: "unrecognized value is passed through", region: "listener-xx.logz.io", want: "listener-xx.logz.io"},
		{name: "empty", region: "", want: ""},
	}

	l := &Logzio{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, l.normalizeRegion(tt.region))
		})
	}
}

func TestLogzioModifyConfigRegion(t *testing.T) {
	tests := []struct {
		name             string
		region           string
		wantExporterCode string
		wantMetricsUrl   string
	}{
		{
			name:             "eu dropdown value reaches the eu listener",
			region:           "listener-eu.logz.io",
			wantExporterCode: "eu",
			wantMetricsUrl:   "https://listener-eu.logz.io:8053",
		},
		{
			name:             "uk dropdown value reaches the uk listener",
			region:           "listener-uk.logz.io",
			wantExporterCode: "uk",
			wantMetricsUrl:   "https://listener-uk.logz.io:8053",
		},
		{
			name:             "us dropdown value keeps the us listener",
			region:           "listener.logz.io",
			wantExporterCode: "us",
			wantMetricsUrl:   "https://listener.logz.io:8053",
		},
		{
			name:             "legacy two letter code still works",
			region:           "au",
			wantExporterCode: "au",
			wantMetricsUrl:   "https://listener-au.logz.io:8053",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dest := &mockDestination{
				id:     "test-id",
				config: map[string]string{"LOGZIO_REGION": tt.region},
				signals: []common.ObservabilitySignal{
					common.TracesObservabilitySignal,
					common.MetricsObservabilitySignal,
					common.LogsObservabilitySignal,
				},
			}

			currentConfig := &Config{
				Exporters:  GenericMap{},
				Processors: GenericMap{},
				Service:    Service{Pipelines: map[string]Pipeline{}},
			}

			pipelines, err := (&Logzio{}).ModifyConfig(dest, currentConfig)
			require.NoError(t, err)
			require.Len(t, pipelines, 3)

			// the logzio exporter only understands two letter region codes, anything
			// else silently falls back to the US listener
			traces, ok := currentConfig.Exporters["logzio/tracing-test-id"].(GenericMap)
			require.True(t, ok)
			assert.Equal(t, tt.wantExporterCode, traces["region"])

			logs, ok := currentConfig.Exporters["logzio/logs-test-id"].(GenericMap)
			require.True(t, ok)
			assert.Equal(t, tt.wantExporterCode, logs["region"])

			metrics, ok := currentConfig.Exporters["prometheusremotewrite/logzio-test-id"].(GenericMap)
			require.True(t, ok)
			assert.Equal(t, tt.wantMetricsUrl, metrics["endpoint"])
		})
	}
}
