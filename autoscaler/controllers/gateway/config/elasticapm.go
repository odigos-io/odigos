package config

import (
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/log"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	elasticApmServerEndpoint = "ELASTIC_APM_SERVER_ENDPOINT"
	elasticApmServerToken    = "${ELASTIC_APM_SECRET_TOKEN}"
)

type ElasticAPM struct{}

func (e *ElasticAPM) DestType() common.DestinationType {
	return common.ElasticAPMDestinationType
}

func (e *ElasticAPM) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {
	var isTlsDisabled = false
	if !e.requiredVarsExists(dest) {
		log.Log.V(0).Info("ElasticAPM config is missing required variables")
	}

	isTlsDisabled = strings.Contains(dest.Spec.Data[elasticApmServerEndpoint], "http://")

	elasticApmEndpoint, err := e.parseEndpoint(dest.Spec.Data[elasticApmServerEndpoint])
	if err != nil {
		log.Log.V(0).Info("ElasticAPM endpoint is not a valid")
	}

	currentConfig.Exporters["otlp/elastic"] = commonconf.GenericMap{
		"endpoint": elasticApmEndpoint,
		"tls": commonconf.GenericMap{
			"insecure": isTlsDisabled,
		},
		"headers": commonconf.GenericMap{
			"authorization": "Bearer ${ELASTIC_APM_SECRET_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		currentConfig.Service.Pipelines["traces/elastic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/elastic"},
		}
	}

	if isMetricsEnabled(dest) {
		currentConfig.Service.Pipelines["metrics/elastic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/elastic"},
		}
	}

	if isLoggingEnabled(dest) {
		currentConfig.Service.Pipelines["logs/elastic"] = commonconf.Pipeline{
			Receivers:  []string{"otlp"},
			Processors: []string{"batch"},
			Exporters:  []string{"otlp/elastic"},
		}
	}

}

func (e *ElasticAPM) requiredVarsExists(dest *odigosv1.Destination) bool {
	if _, ok := dest.Spec.Data[elasticApmServerEndpoint]; !ok {
		return false
	}

	if _, ok := dest.Spec.Data[elasticApmServerToken]; !ok {
		return false
	}

	return true
}

func (e *ElasticAPM) parseEndpoint(endpoint string) (string, error) {
	var port = "8200"
	endpoint = strings.Trim(endpoint, "http://")
	endpoint = strings.Trim(endpoint, "https://")
	endpointDetails := strings.Split(endpoint, ":")
	host := endpointDetails[0]
	if len(endpointDetails) > 1 {
		port = endpointDetails[1]
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}
