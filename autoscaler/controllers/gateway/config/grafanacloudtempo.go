package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
)

const (
	grafanaCloudTempoEndpointKey = "GRAFANA_CLOUD_TEMPO_ENDPOINT"
	grafanaCloudTempoUsernameKey = "GRAFANA_CLOUD_TEMPO_USERNAME"
)

type GrafanaCloudTempo struct{}

func (g *GrafanaCloudTempo) DestType() common.DestinationType {
	return common.GrafanaCloudTempoDestinationType
}

func (g *GrafanaCloudTempo) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {

	if !isTracingEnabled(dest) {
		return errors.New("Tracing not enabled, gateway will not be configured for grafana cloud Tempo")
	}

	tempoUrl, exists := dest.Spec.Data[grafanaCloudTempoEndpointKey]
	if !exists {
		return errors.New("Grafana Cloud Tempo endpoint not specified, gateway will not be configured for Tempo")
	}

	tempoUsername, exists := dest.Spec.Data[grafanaCloudTempoUsernameKey]
	if !exists {
		return errors.New("Grafana Cloud Tempo username not specified, gateway will not be configured for Tempo")
	}

	grpcEndpointUrl := grafanaTempoUrlFromInput(tempoUrl)

	authExtensionName := "basicauth/grafana" + dest.Name
	currentConfig.Extensions[authExtensionName] = commonconf.GenericMap{
		"client_auth": commonconf.GenericMap{
			"username": tempoUsername,
			"password": "${GRAFANA_CLOUD_TEMPO_PASSWORD}",
		},
	}

	exporterName := "otlp/grafana-" + dest.Name
	currentConfig.Exporters[exporterName] = commonconf.GenericMap{
		"endpoint": grpcEndpointUrl,
		"auth": commonconf.GenericMap{
			"authenticator": authExtensionName,
		},
	}

	tracesPipelineName := "traces/grafana-" + dest.Name
	currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
		Exporters: []string{exporterName},
	}

	return nil
}

// grafana cloud tempo url for otlp grpc should be of the form:
// tempo-prod-10-prod-eu-west-2.grafana.net:443
//
// if one uses tempo as a grafana datasource, the url for the datasource will be of the form:
// https://tempo-prod-10-prod-eu-west-2.grafana.net/tempo
// we will accept both forms as input
func grafanaTempoUrlFromInput(rawUrl string) string {
	otlpEndpointUrl := rawUrl
	otlpEndpointUrl = strings.TrimPrefix(otlpEndpointUrl, "https://")
	otlpEndpointUrl = strings.TrimSuffix(otlpEndpointUrl, "/tempo")

	if !strings.Contains(otlpEndpointUrl, ":") {
		otlpEndpointUrl = fmt.Sprintf("%s:%d", otlpEndpointUrl, 443)
	}

	return otlpEndpointUrl
}
