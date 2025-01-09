package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	JaegerUrlKey   = "JAEGER_URL"
	JaegerTlsKey   = "JAEGER_TLS_ENABLED"
	JaegerCaPemKey = "JAEGER_CA_PEM"
)

var (
	ErrorJaegerMissingURL        = errors.New("Jaeger is missing a required field (\"JAEGER_URL\"), Jaeger will not be configured")
	ErrorJaegerTracingDisabled   = errors.New("Jaeger is missing a required field (\"TRACES\"), Jaeger will not be configured")
	ErrorJaegerMetricsNotAllowed = errors.New("Jaeger has a forbidden field (\"METRICS\"), Jaeger will not be configured")
	ErrorJaegerLogsNotAllowed    = errors.New("Jaeger has a forbidden field (\"LOGS\"), Jaeger will not be configured")
)

type Jaeger struct{}

// compile time checks
var _ Configer = (*Jaeger)(nil)

func (j *Jaeger) DestType() common.DestinationType {
	return common.JaegerDestinationType
}

func (j *Jaeger) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	config := dest.GetConfig()
	uniqueUri := "jaeger-" + dest.GetID()

	url, urlExists := config[JaegerUrlKey]
	if !urlExists {
		return nil, ErrorJaegerMissingURL
	}

	tls := dest.GetConfig()[JaegerTlsKey]
	tlsEnabled := tls == "true"

	endpoint, err := parseOtlpGrpcUrl(url, tlsEnabled)
	if err != nil {
		return nil, err
	}

	exporterName := "otlp/" + uniqueUri
	exporterConfig := GenericMap{
		"endpoint": endpoint,
	}
	tlsConfig := GenericMap{
		"insecure": !tlsEnabled,
	}
	caPem, caExists := dest.GetConfig()[JaegerCaPemKey]
	if caExists && caPem != "" {
		tlsConfig["ca_pem"] = caPem
	}

	exporterConfig["tls"] = tlsConfig
	currentConfig.Exporters[exporterName] = exporterConfig
	pipelineNames := []string{}
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	} else {
		return nil, ErrorJaegerTracingDisabled
	}

	if isMetricsEnabled(dest) {
		return nil, ErrorJaegerMetricsNotAllowed
	}

	if isLoggingEnabled(dest) {
		return nil, ErrorJaegerLogsNotAllowed
	}

	return pipelineNames, nil
}
