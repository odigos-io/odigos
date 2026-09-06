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

// remoteWriteEndpoint returns the remote write endpoint to configure.
// If the url already ends with a known remote write path (Prometheus/Thanos
// use /api/v1/write, Mimir uses /api/v1/push) it is used as-is, allowing users
// to specify the full endpoint. Otherwise the standard /api/v1/write suffix is
// appended for backwards compatibility with hosts provided without a path.
func remoteWriteEndpoint(url string) string {
	for _, suffix := range []string{"/api/v1/write", "/api/v1/push"} {
		if strings.HasSuffix(url, suffix) {
			return url
		}
	}
	return fmt.Sprintf("%s/api/v1/write", url)
}

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
	endpoint := remoteWriteEndpoint(url)
	rwExporterName := "prometheusremotewrite/" + uniqueUri

	exporterConfig := GenericMap{
		"endpoint": endpoint,
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
