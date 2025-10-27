package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	lokiUrlKey      = "LOKI_URL"
	lokiLabelsKey   = "LOKI_LABELS"
	lokiUsernameKey = "LOKI_USERNAME"
	lokiPasswordKey = "LOKI_PASSWORD"
	lokiApiPath     = "/otlp"
)

type Loki struct{}

func (l *Loki) DestType() common.DestinationType {
	return common.LokiDestinationType
}

func (l *Loki) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !isLoggingEnabled(dest) {
		return nil, errors.New("logging not enabled, gateway will not be configured for Loki")
	}

	destConfig := dest.GetConfig()
	uniqueUri := "loki-" + dest.GetID()

	rawUrl, exists := destConfig[lokiUrlKey]
	if !exists {
		return nil, errors.New("Loki endpoint not specified, gateway will not be configured for Loki")
	}

	lokiUrl, err := lokiUrlFromInput(rawUrl)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to parse loki endpoint, gateway will not be configured for Loki"))
	}

	rawLokiLabels, exists := destConfig[lokiLabelsKey]
	lokiProcessors, err := lokiLabelsProcessors(rawLokiLabels, exists, uniqueUri)
	if err != nil {
		return nil, errors.Join(err, errors.New("failed to parse loki labels, gateway will not be configured for Loki"))
	}

	exporterConf := GenericMap{
		"endpoint": lokiUrl,
	}

	lokiUsername, usernameExists := destConfig[lokiUsernameKey]
	// Check if basic auth is configured (optional for self-hosted Loki)
	if usernameExists && lokiUsername != "" {
		// Add basic auth if username is provided
		authExtensionName := "basicauth/" + uniqueUri
		currentConfig.Extensions[authExtensionName] = GenericMap{
			"client_auth": GenericMap{
				"username": lokiUsername,
				"password": fmt.Sprintf("${%s}", lokiPasswordKey),
			},
		}
		exporterConf["auth"] = GenericMap{
			"authenticator": authExtensionName,
		}
		currentConfig.Service.Extensions = append(currentConfig.Service.Extensions, authExtensionName)
	}

	lokiExporterName := "otlphttp/" + uniqueUri
	currentConfig.Exporters[lokiExporterName] = exporterConf

	processorNames := []string{}
	for k, v := range lokiProcessors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	logsPipelineName := "logs/" + uniqueUri
	currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
		Processors: processorNames,
		Exporters:  []string{lokiExporterName},
	}

	return []string{logsPipelineName}, nil
}

func lokiUrlFromInput(rawUrl string) (string, error) {
	rawUrl = strings.TrimSpace(rawUrl)
	urlWithScheme := rawUrl

	// since loki is a self hosted destination, we will fallback to http if the scheme is not provided
	if !strings.Contains(rawUrl, "://") {
		urlWithScheme = "http://" + rawUrl
	}

	parsedUrl, err := url.Parse(urlWithScheme)
	if err != nil {
		return "", err
	}

	if parsedUrl.Scheme != "http" && parsedUrl.Scheme != "https" {
		return "", fmt.Errorf("loki endpoint scheme must be http or https got %s", parsedUrl.Scheme)
	}

	// we allow the user to specify the path, but will fallback to the default loki path if not provided
	if parsedUrl.Path == "" {
		parsedUrl.Path = lokiApiPath
	}

	// we allow the user to specify the port, but will fallback to the default loki port if not provided
	if parsedUrl.Port() == "" {
		if parsedUrl.Host == "" {
			return "", fmt.Errorf("loki endpoint host is required")
		}
		parsedUrl.Host += ":3100"
	}

	return parsedUrl.String(), nil
}

// odigos handles log records in otel format, e.g. with resource and log attributes.
// loki architecture works with labels, where each combination of labels values is a stream.
// This function creates processors to convert otel attributes to loki labels based on the user configuration.
func lokiLabelsProcessors(rawLabels string, exists bool, destName string) (GenericMap, error) {
	// backwards compatibility, if the user labels are not provided, we use the default
	if !exists {
		processorName := "attributes/" + destName
		return GenericMap{
			processorName: GenericMap{
				"actions": []GenericMap{
					{
						"key":    "loki.attribute.labels",
						"action": "insert",
						"value":  "k8s.container.name, k8s.pod.name, k8s.namespace.name",
					},
				},
			},
		}, nil
	}

	// no labels. not recommended, but ok
	if rawLabels == "" || rawLabels == "[]" {
		return GenericMap{}, nil
	}

	var attributeNames []string
	err := json.Unmarshal([]byte(rawLabels), &attributeNames)
	if err != nil {
		return nil, err
	}
	attributeHint := strings.Join(attributeNames, ", ")

	processors := GenericMap{}

	// since we don't know if the attributes are logs attributes or resource attributes, we will add them to both processors
	attributesProcessorName := "attributes/" + destName
	processors[attributesProcessorName] = GenericMap{
		"actions": []GenericMap{
			{
				"key":    "loki.attribute.labels",
				"action": "insert",
				"value":  attributeHint,
			},
		},
	}

	resourceProcessorName := "resource/" + destName
	processors[resourceProcessorName] = GenericMap{
		"attributes": []GenericMap{
			{
				"key":    "loki.resource.labels",
				"action": "insert",
				"value":  attributeHint,
			},
		},
	}

	return processors, nil
}
