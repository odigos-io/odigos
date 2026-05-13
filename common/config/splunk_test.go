package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type splunkTestDestination struct {
	id      string
	signals []common.ObservabilitySignal
	cfg     map[string]string
}

func (d splunkTestDestination) GetID() string {
	return d.id
}

func (d splunkTestDestination) GetType() common.DestinationType {
	return common.SplunkDestinationType
}

func (d splunkTestDestination) GetConfig() map[string]string {
	return d.cfg
}

func (d splunkTestDestination) GetSignals() []common.ObservabilitySignal {
	return d.signals
}

func TestSplunkModifyConfigUsesOTLPHTTPExporter(t *testing.T) {
	s := &Splunk{}
	cfg := &Config{
		Exporters: GenericMap{},
		Service: Service{
			Pipelines: map[string]Pipeline{},
		},
	}

	dest := splunkTestDestination{
		id:      "dest1",
		signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		cfg: map[string]string{
			splunkRealm: "us1",
		},
	}

	pipelines, err := s.ModifyConfig(dest, cfg)
	require.NoError(t, err)
	require.Equal(t, []string{"traces/splunk-dest1"}, pipelines)

	exporterName := "otlphttp/dest1"
	require.Contains(t, cfg.Exporters, exporterName)

	exporterCfg, ok := cfg.Exporters[exporterName].(GenericMap)
	require.True(t, ok)
	assert.Equal(t, "https://ingest.us1.signalfx.com/v2/trace/otlp", exporterCfg["traces_endpoint"])

	headers, ok := exporterCfg["headers"].(GenericMap)
	require.True(t, ok)
	assert.Equal(t, "${SPLUNK_ACCESS_TOKEN}", headers["X-SF-Token"])

	require.Contains(t, cfg.Service.Pipelines, "traces/splunk-dest1")
	assert.Equal(t, []string{exporterName}, cfg.Service.Pipelines["traces/splunk-dest1"].Exporters)
}

func TestSplunkModifyConfigRequiresRealm(t *testing.T) {
	s := &Splunk{}
	cfg := &Config{
		Exporters: GenericMap{},
		Service: Service{
			Pipelines: map[string]Pipeline{},
		},
	}

	dest := splunkTestDestination{
		id:      "dest1",
		signals: []common.ObservabilitySignal{common.TracesObservabilitySignal},
		cfg:     map[string]string{},
	}

	_, err := s.ModifyConfig(dest, cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Splunk realm not specified")
}
