package config

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
)

const (
	signalfxRealm              = "SIGNALFX_REALM"
	signalfxCaPem              = "SIGNALFX_CA_PEM"
	signalfxInsecureSkipVerify = "SIGNALFX_INSECURE_SKIP_VERIFY"

	// SignalfxCaMountPath is the path where the CA certificate is mounted in the collector pod
	SignalfxCaMountPath        = "/etc/signalfx/certs"
	SignalfxCaSecretVolumeName = "signalfx-ca-cert"
)

type SignalFx struct{}

func (s *SignalFx) DestType() common.DestinationType {
	return common.SignalFxDestinationType
}

func (s *SignalFx) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	realm := config[signalfxRealm]

	var pipelineNames []string

	if isMetricsEnabled(dest) {
		exporterName := "signalfx/" + dest.GetID()
		exporterConfig := GenericMap{
			"access_token": SecretEnvPlaceholder("SIGNALFX_ACCESS_TOKEN", dest),
			"realm":        realm,
			"api_url":      fmt.Sprintf("https://api.%s.signalfx.com", realm),
			"ingest_url":   fmt.Sprintf("https://ingest.%s.signalfx.com", realm),
		}
		if tlsConfig := signalfxTLSConfig(config); tlsConfig != nil {
			exporterConfig["tls"] = tlsConfig
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		pipelineName := "metrics/signalfx-" + dest.GetID()
		currentConfig.Service.Pipelines[pipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipelineName)
	}

	// The signalfx exporter handles metrics and events only, so traces go to the same
	// Splunk Observability Cloud realm over OTLP/HTTP, as the legacy splunk destination does.
	if isTracingEnabled(dest) {
		exporterName := "otlp_http/signalfx-" + dest.GetID()
		exporterConfig := GenericMap{
			"traces_endpoint": fmt.Sprintf("https://ingest.%s.signalfx.com/v2/trace/otlp", realm),
			"headers": GenericMap{
				"X-SF-Token": SecretEnvPlaceholder("SIGNALFX_ACCESS_TOKEN", dest),
			},
		}
		if tlsConfig := signalfxTLSConfig(config); tlsConfig != nil {
			exporterConfig["tls"] = tlsConfig
		}
		currentConfig.Exporters[exporterName] = exporterConfig

		pipelineName := "traces/signalfx-" + dest.GetID()
		currentConfig.Service.Pipelines[pipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, pipelineName)
	}

	return pipelineNames, nil
}

// signalfxTLSConfig returns the exporter tls block when the destination customizes TLS, and nil
// otherwise. A fresh map is returned on every call so exporters never share one.
func signalfxTLSConfig(config map[string]string) GenericMap {
	caPem := config[signalfxCaPem]
	insecureSkipVerify := config[signalfxInsecureSkipVerify] == "true"

	if caPem == "" && !insecureSkipVerify {
		return nil
	}

	tlsConfig := GenericMap{"insecure_skip_verify": insecureSkipVerify}
	if caPem != "" {
		tlsConfig["ca_file"] = SignalfxCaMountPath + "/" + signalfxCaPem
	}
	return tlsConfig
}
