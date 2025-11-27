package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	promRWurlKey             = "PROMETHEUS_REMOTEWRITE_URL"
	promAuthHeaderKey        = "PROMETHEUS_BEARER_TOKEN"
	promBasicAuthUsernameKey = "PROMETHEUS_BASIC_AUTH_USERNAME"
	promBasicAuthPasswordKey = "PROMETHEUS_BASIC_AUTH_PASSWORD"
)

type Prometheus struct{}

func (p *Prometheus) DestType() common.DestinationType {
	return common.PrometheusDestinationType
}

func (p *Prometheus) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	uniqueUri := "prometheus-" + dest.GetID()
	config := dest.GetConfig()

	url, exists := config[promRWurlKey]
	if !exists {
		return nil, errors.New("Prometheus remote writer url not specified, gateway will not be configured for prometheus")
	}

	if !isMetricsEnabled(dest) {
		return nil, errors.New("metrics not enabled for prometheus destination, gateway will not be configured for prometheus")
	}

	url = addProtocol(url)
	url = strings.TrimSuffix(url, "/api/v1/write")
	rwExporterName := "prometheusremotewrite/" + uniqueUri

	exporterConfig := GenericMap{
		"endpoint": fmt.Sprintf("%s/api/v1/write", url),
		"resource_to_telemetry_conversion": GenericMap{
			"enabled": true,
		},
		"headers": GenericMap{
			"Authorization": "Bearer ${PROMETHEUS_BEARER_TOKEN}",
		},
	}

	// Check for Bearer token or Basic Auth (Bearer token takes precedence)
	bearerToken, bearerExists := config[promAuthHeaderKey]
	username, usernameExists := config[promBasicAuthUsernameKey]

	if bearerExists && bearerToken != "" {
		// Use Bearer token authentication via headers
		fmt.Printf("========== Using Bearer token authentication\n", bearerToken)
		exporterConfig["headers"] = GenericMap{
			"Authorization": "Bearer ${PROMETHEUS_BEARER_TOKEN}",
		}
	} else if usernameExists && username != "" {
		// Use Basic Auth via authenticator extension
		fmt.Printf("========== Using Basic Auth authentication\n", username, promBasicAuthPasswordKey)
		authExtensionName := "basicauth/" + uniqueUri
		currentConfig.Extensions[authExtensionName] = GenericMap{
			"client_auth": GenericMap{
				"username": username,
				"password": fmt.Sprintf("${%s}", promBasicAuthPasswordKey),
			},
		}
		exporterConfig["auth"] = GenericMap{
			"authenticator": authExtensionName,
		}
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	}

	currentConfig.Exporters[rwExporterName] = exporterConfig

	metricsPipelineName := "metrics/" + uniqueUri
	currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
		Exporters: []string{rwExporterName},
	}

	return []string{metricsPipelineName}, nil
}
