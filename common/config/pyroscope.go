package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

const (
	pyroscopeUrlKey = "PYROSCOPE_URL"
	pyroscopeTlsKey = "PYROSCOPE_TLS_ENABLED"
)

var (
	ErrorPyroscopeMissingURL       = errors.New("Pyroscope is missing a required field (\"PYROSCOPE_URL\"), Pyroscope will not be configured")
	ErrorPyroscopeProfilesDisabled = errors.New("Pyroscope requires PROFILES signal to be enabled")
)

type Pyroscope struct{}

var _ Configer = (*Pyroscope)(nil)

func (p *Pyroscope) DestType() common.DestinationType {
	return common.PyroscopeDestinationType
}

func (p *Pyroscope) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isProfilingEnabled(dest) {
		return nil, ErrorPyroscopeProfilesDisabled
	}

	cfg := dest.GetConfig()

	url, exists := cfg[pyroscopeUrlKey]
	if !exists || url == "" {
		return nil, ErrorPyroscopeMissingURL
	}

	tlsEnabled := cfg[pyroscopeTlsKey] == "true"
	scheme := "http"
	if tlsEnabled {
		scheme = "https"
	}

	// Grafana Pyroscope registers HTTP OTLP profiles at /v1development/profiles on the
	// server root (see RegisterDistributor in grafana/pyroscope). The OpenTelemetry
	// otlphttp exporter appends /v1development/profiles to this base URL — it must not
	// include a /otlp prefix or requests hit .../otlp/v1development/profiles (404).
	baseEndpoint := fmt.Sprintf("%s://%s", scheme, url)

	exporterName := "otlphttp/pyroscope-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": baseEndpoint,
		"tls": GenericMap{
			"insecure": !tlsEnabled,
		},
	}

	addProfilesPipeline(currentConfig, "pyroscope", dest.GetID(), exporterName)

	return nil, nil
}
