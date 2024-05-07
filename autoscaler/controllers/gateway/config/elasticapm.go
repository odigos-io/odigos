package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	elasticApmServerEndpoint = "ELASTIC_APM_SERVER_ENDPOINT"
	elasticApmServerToken    = "${ELASTIC_APM_SECRET_TOKEN}"
)

type ElasticAPM struct{}

func (e *ElasticAPM) DestType() common.DestinationType {
	return common.ElasticAPMDestinationType
}

func (e *ElasticAPM) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {
	var isTlsDisabled = false
	if !e.requiredVarsExists(dest) {
		return errors.New("ElasticAPM config is missing required variables")
	}

	isTlsDisabled = strings.Contains(dest.Spec.Data[elasticApmServerEndpoint], "http://")

	elasticApmEndpoint, err := e.parseEndpoint(dest.Spec.Data[elasticApmServerEndpoint])
	if err != nil {
		return errors.Join(err, errors.New("ElasticAPM endpoint is not a valid"))
	}

	exporterName := "otlp/elastic-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": elasticApmEndpoint,
		"tls": commonconf.GenericMap{
			"insecure": isTlsDisabled,
		},
		"headers": commonconf.GenericMap{
			"authorization": "Bearer ${ELASTIC_APM_SECRET_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/elastic-" + dest.Name
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/elastic-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/elastic-" + dest.Name
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
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
