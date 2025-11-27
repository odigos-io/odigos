package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	promRWurlKey             = "PROMETHEUS_REMOTEWRITE_URL"
	promUseAuthenticationKey = "PROMETHEUS_USE_AUTHENTICATION"
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
	}

	// Check if authentication is enabled
	useAuth := config[promUseAuthenticationKey] == "true"
	username, usernameExists := config[promBasicAuthUsernameKey]

	// Only configure auth if explicitly enabled
	if useAuth {
		// Ensure Extensions map is initialized
		if currentConfig.Extensions == nil {
			currentConfig.Extensions = GenericMap{}
		}

		if usernameExists && username != "" {
			// Use Basic Auth via authenticator extension
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
		} else {
			// Use Bearer token authentication via authenticator extension
			// Token may be in config map or in a Secret (injected as env var)
			authExtensionName := "bearertokenauth/" + uniqueUri
			currentConfig.Extensions[authExtensionName] = GenericMap{
				"token": fmt.Sprintf("${%s}", promAuthHeaderKey),
			}
			exporterConfig["auth"] = GenericMap{
				"authenticator": authExtensionName,
			}
			currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
		}
	}
	// If authentication is not enabled, no auth extensions are configured

	currentConfig.Exporters[rwExporterName] = exporterConfig

	metricsPipelineName := "metrics/" + uniqueUri
	currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
		Exporters: []string{rwExporterName},
	}

	return []string{metricsPipelineName}, nil
}
