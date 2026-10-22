package config

import (
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/common"
)

const (
	splunkRealm                  = "SPLUNK_REALM"
	splunkOtlpTlsKey             = "SPLUNK_OTLP_TLS_ENABLED"
	splunkOtlpCaPemKey           = "SPLUNK_OTLP_CA_PEM"
	splunkOtlpInsecureSkipVerify = "SPLUNK_OTLP_INSECURE_SKIP_VERIFY"
	splunkOtlpCompression        = "SPLUNK_OTLP_COMPRESSION"
)

// Splunk preserves the legacy destination type using OTLP/HTTP after SAPM was removed upstream.
//
// Deprecated: Use SplunkOTLP for new destinations.
type Splunk struct{}

func (s *Splunk) DestType() common.DestinationType {
	return common.SplunkDestinationType
}

func (s *Splunk) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	return (&SplunkOTLP{}).modifyConfig(dest, currentConfig, "splunk")
}

// SplunkOTLP configures an OTLPHTTP exporter configured for Splunk ingestion.
type SplunkOTLP struct{}

func (s *SplunkOTLP) DestType() common.DestinationType {
	return common.SplunkOTLPDestinationType
}

func (s *SplunkOTLP) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	return s.modifyConfig(dest, currentConfig, "splunkotlp")
}

func (s *SplunkOTLP) modifyConfig(dest ExporterConfigurer, currentConfig *Config, pipelinePrefix string) ([]string, error) {
	config := dest.GetConfig()

	realm, exists := config[splunkRealm]
	if !exists {
		return nil, errors.New("Splunk realm not specified, gateway will not be configured for Splunk")
	}

	tls := config[splunkOtlpTlsKey]
	tlsEnabled := tls == "true"
	tlsConfig := GenericMap{
		"insecure": !tlsEnabled,
	}
	caPem, caExists := config[splunkOtlpCaPemKey]
	if caExists && caPem != "" {
		tlsConfig["ca_pem"] = caPem
	}
	insecureSkipVerify, skipExists := config[splunkOtlpInsecureSkipVerify]
	if skipExists && insecureSkipVerify != "" {
		tlsConfig["insecure_skip_verify"] = parseBool(insecureSkipVerify)
	}

	var pipelineNames []string
	if isTracingEnabled(dest) {
		exporterName := "otlp_http/" + dest.GetID()
		exporterConf := GenericMap{
			"headers": GenericMap{
				"X-SF-Token": SecretEnvPlaceholder("SPLUNK_ACCESS_TOKEN", dest),
			},
			"traces_endpoint": fmt.Sprintf("https://ingest.%s.signalfx.com/v2/trace/otlp", realm),
		}
		if compression, ok := config[splunkOtlpCompression]; ok {
			exporterConf["compression"] = compression
		}
		if tlsEnabled {
			exporterConf["tls"] = tlsConfig
		}

		currentConfig.Exporters[exporterName] = exporterConf

		tracesPipelineName := "traces/" + pipelinePrefix + "-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	return pipelineNames, nil
}
