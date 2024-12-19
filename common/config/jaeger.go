package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	JaegerUrlKey   = "JAEGER_URL"
	JaegerTlsKey   = "JAEGER_TLS"
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

func (j *Jaeger) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	config := dest.GetConfig()
	uniqueUri := "jaeger-" + dest.GetID()

	url, urlExists := config[JaegerUrlKey]
	if !urlExists {
		return ErrorJaegerMissingURL
	}

	exporterName := "otlp/" + uniqueUri
	var exporterConfig GenericMap

	tls, tlsExists := dest.GetConfig()[JaegerTlsKey]
	if tlsExists && tls == "true" {
		// Will use a secure connection with TLS over GRPC
		endpoint, err := parseOtlpGrpcUrl(url, true)
		if err != nil {
			return err
		}

		exporterConfig = GenericMap{
			"endpoint": endpoint,
		}
		tlsConfig := GenericMap{
			"insecure": false,
		}

		caPem, caExists := dest.GetConfig()[JaegerCaPemKey]
		if caExists && caPem != "" {
			tlsConfig["ca_pem"] = caPem
		}

		exporterConfig["tls"] = tlsConfig
	} else {
		// Will use an insecure connection over GRPC
		endpoint, err := parseOtlpGrpcUrl(url, false)
		if err != nil {
			return err
		}

		exporterConfig = GenericMap{
			"endpoint": endpoint,
			"tls": GenericMap{
				"insecure": true,
			},
		}
	}

	currentConfig.Exporters[exporterName] = exporterConfig

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/" + uniqueUri
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	} else {
		return ErrorJaegerTracingDisabled
	}

	if isMetricsEnabled(dest) {
		return ErrorJaegerMetricsNotAllowed
	}

	if isLoggingEnabled(dest) {
		return ErrorJaegerLogsNotAllowed
	}

	return nil
}
