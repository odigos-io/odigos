package config

import (
	"errors"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
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

func (j *Jaeger) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	if !isTracingEnabled(dest) {
		log.Log.V(0).Error(ErrorJaegerTracingDisabled, "skipping Jaeger destination config")
		return
	}

	url, urlExist := dest.Spec.Data[jaegerUrlKey]
	if !urlExist {
		log.Log.V(0).Error(ErrorJaegerMissingURL, "skipping Jaeger destination config")
		return
	}

	grpcEndpoint, err := parseUnencryptedOtlpGrpcUrl(url)
	if err != nil {
		log.Log.V(0).Error(err, "skipping Jaeger destination config")
		return
	}

	exporterName := "otlp/jaeger-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": grpcEndpoint,
		"tls": commonconf.GenericMap{
			"insecure": true,
		},
	}

	pipelineName := "traces/jaeger-" + dest.Name
	currentConfig.Service.Pipelines[pipelineName] = commonconf.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: []string{"batch"},
		Exporters:  []string{exporterName},
	}
}
