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

	exporterName := "signalfx/" + dest.GetID()
	exporterConfig := GenericMap{
		"access_token": "${SIGNALFX_ACCESS_TOKEN}",
		"realm":        realm,
		"api_url":      fmt.Sprintf("https://api.%s.signalfx.com", realm),
		"ingest_url":   fmt.Sprintf("https://ingest.%s.signalfx.com", realm),
	}

	// Add TLS config if CA PEM or insecure skip verify is set
	caPem := config[signalfxCaPem]
	insecureSkipVerify := config[signalfxInsecureSkipVerify] == "true"

	if caPem != "" || insecureSkipVerify {
		tlsConfig := GenericMap{}
		if caPem != "" {
			tlsConfig["ca_file"] = SignalfxCaMountPath + "/" + signalfxCaPem
		}
		if insecureSkipVerify {
			tlsConfig["insecure_skip_verify"] = true
		} else {
			tlsConfig["insecure_skip_verify"] = false
		}
		exporterConfig["tls"] = tlsConfig
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	// Configure metrics pipeline only
	pipelineName := "metrics/signalfx-" + dest.GetID()
	currentConfig.Service.Pipelines[pipelineName] = Pipeline{
		Exporters: []string{exporterName},
	}

	return []string{pipelineName}, nil
}
