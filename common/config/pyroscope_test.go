package config

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	"github.com/stretchr/testify/require"
)

type pyroscopeTestDest struct {
	id     string
	cfg    map[string]string
	signal []common.ObservabilitySignal
}

func (d pyroscopeTestDest) GetID() string                              { return d.id }
func (d pyroscopeTestDest) GetType() common.DestinationType            { return common.PyroscopeDestinationType }
func (d pyroscopeTestDest) GetConfig() map[string]string               { return d.cfg }
func (d pyroscopeTestDest) GetSignals() []common.ObservabilitySignal {
	return d.signal
}

// Pyroscope serves OTLP profiles at /v1development/profiles on the server root.
// The OTLP HTTP exporter appends that path to the configured endpoint base URL.
// A base of scheme://host:port/otlp would post to .../otlp/v1development/profiles (404).
func TestPyroscopeModifyConfig_OtlpHttpEndpointIsHostRoot(t *testing.T) {
	p := &Pyroscope{}
	dest := pyroscopeTestDest{
		id: "d1",
		cfg: map[string]string{
			pyroscopeUrlKey: "pyroscope.pyroscope:4040",
			pyroscopeTlsKey: "false",
		},
		signal: []common.ObservabilitySignal{common.ProfilesObservabilitySignal},
	}
	cfg := &Config{Exporters: GenericMap{}, Service: Service{Pipelines: map[string]Pipeline{}}}
	_, err := p.ModifyConfig(dest, cfg)
	require.NoError(t, err)

	exp := cfg.Exporters["otlphttp/pyroscope-d1"].(GenericMap)
	ep, ok := exp["endpoint"].(string)
	require.True(t, ok)
	require.Equal(t, "http://pyroscope.pyroscope:4040", ep)
	require.NotContains(t, ep, "/otlp")
}
