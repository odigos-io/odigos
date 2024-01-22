package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
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
		ctrl.Log.Error(ErrorJaegerTracingDisabled, "skipping Jaeger destination config")
		return
	}

	url, urlExist := dest.Spec.Data[jaegerUrlKey]
	if !urlExist {
		ctrl.Log.Error(ErrorJaegerMissingURL, "skipping Jaeger destination config")
		return
	}

	if strings.HasPrefix(url, "https://") {
		ctrl.Log.Error(ErrorJaegerMissingURL, "skipping Jaeger destination config")
		return
	}

	// no need for the http:// prefix with grpc protocol in golang
	url = strings.TrimPrefix(url, "http://")

	// Check if url does not contains port
	if !strings.Contains(url, ":") {
		url = fmt.Sprintf("%s:4317", url)
	}

	jaegerExporterName := "otlp/jaeger-" + dest.Name
	currentConfig.Exporters[jaegerExporterName] = commonconf.GenericMap{
		"endpoint": url,
		"tls": commonconf.GenericMap{
			"insecure": true,
		},
	}

	pipelineName := "traces/jaeger-" + dest.Name
	currentConfig.Service.Pipelines[pipelineName] = commonconf.Pipeline{
		Receivers:  []string{"otlp"},
		Processors: []string{"batch"},
		Exporters:  []string{jaegerExporterName},
	}
}
