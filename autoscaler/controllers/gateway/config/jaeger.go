package config

import (
	"errors"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

var (
	ErrorJaegerTracingDisabled = errors.New("attempting to configure Jaeger tracing, but tracing is disabled")
	ErrorJaegerMissingURL      = errors.New("missing Jaeger JAEGER_URL config")
	ErrorJaegerNoTls           = errors.New("jaeger destination only supports non tls connections")
)

const (
	jaegerUrlKey = "JAEGER_URL"
)

type Jaeger struct{}

func (j *Jaeger) DestType() common.DestinationType {
	return common.JaegerDestinationType
}

func (j *Jaeger) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {

	if !isTracingEnabled(dest) {
		return ErrorJaegerTracingDisabled
	}

	url, urlExist := dest.GetConfig()[jaegerUrlKey]
	if !urlExist {
		return ErrorJaegerMissingURL
	}

	grpcEndpoint, err := parseUnencryptedOtlpGrpcUrl(url)
	if err != nil {
		return err
	}

	exporterName := "otlp/jaeger-" + dest.GetName()
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": grpcEndpoint,
		"tls": commonconf.GenericMap{
			"insecure": true,
		},
	}

	pipelineName := "traces/jaeger-" + dest.GetName()
	currentConfig.Service.Pipelines[pipelineName] = commonconf.Pipeline{
		Exporters: []string{exporterName},
	}
	return nil
}
