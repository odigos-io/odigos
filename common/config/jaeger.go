package config

import (
	"errors"

	"github.com/odigos-io/odigos/common"
)

const (
	JaegerUrlKey     = "JAEGER_URL"
	JaegerCertPemKey = "JAEGER_CERT_PEM"
	JaegerKeyPemKey  = "JAEGER_KEY_PEM"
	JaegerCaPemKey   = "JAEGER_CA_PEM"
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

	var exporterName string
	var exporterConfig GenericMap

	certPem, certExists := dest.GetConfig()[JaegerCertPemKey]
	keyPem, keyExists := dest.GetConfig()[JaegerKeyPemKey]
	if certExists && keyExists {
		// Client cert & key were found, we will use a secure connection with TLS over GRPC
		endpoint, err := parseEncryptedOtlpGrpcUrl(url)
		if err != nil {
			return err
		}

		exporterName = "otlp/" + uniqueUri
		exporterConfig = GenericMap{
			"endpoint": endpoint,
		}
		tlsConfig := GenericMap{
			"cert_pem": certPem,
			"key_pem":  keyPem,
		}

		caPem, caExists := dest.GetConfig()[JaegerCaPemKey]
		if caExists {
			// CA cert was found, we will include it to allow self-signed certificates to be used
			tlsConfig["ca_pem"] = caPem
		}

		exporterConfig["tls"] = tlsConfig
	} else {
		// Client cert & key were not found, we will use an insecure connection over GRPC
		endpoint, err := parseUnencryptedOtlpGrpcUrl(url)
		if err != nil {
			return err
		}

		exporterName = "otlp/" + uniqueUri
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
