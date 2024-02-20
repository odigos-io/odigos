package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	lokiUrlKey    = "LOKI_URL"
	lokiLabelsKey = "LOKI_LABELS"
)

type Loki struct{}

func (l *Loki) DestType() common.DestinationType {
	return common.LokiDestinationType
}

func (l *Loki) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	rawUrl, exists := dest.Spec.Data[lokiUrlKey]
	if !exists {
		log.Log.V(0).Info("Loki endpoint not specified, gateway will not be configured for Loki")
		return
	}

	url, err := lokiUrlFromInput(rawUrl)
	if err != nil {
		log.Log.V(0).Error(err, "failed to parse loki endpoint, gateway will not be configured for Loki")
		return
	}

	rawLokiLabels, exists := dest.Spec.Data[lokiLabelsKey]
	lokiProcessors, err := lokiLabelsProcessors(rawLokiLabels, exists, dest.Name)
	if err != nil {
		log.Log.V(0).Error(err, "failed to parse loki labels, gateway will not be configured for Loki")
		return
	}

	lokiExporterName := "loki/loki-" + dest.Name
	currentConfig.Exporters[lokiExporterName] = commonconf.GenericMap{
		"endpoint": url,
	}

	processorNames := []string{}
	for k, v := range lokiProcessors {
		currentConfig.Processors[k] = v
		processorNames = append(processorNames, k)
	}

	logsPipelineName := "logs/loki-" + dest.Name
	currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
		Processors: processorNames,
		Exporters:  []string{lokiExporterName},
	}
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
		parsedUrl.Path = "/loki/api/v1/push"
	}

	// we allow the user to specify the port, but will fallback to the default loki port if not provided
	if parsedUrl.Port() == "" {
		if parsedUrl.Host == "" {
			return "", fmt.Errorf("loki endpoint host is required")
		}
		parsedUrl.Host = parsedUrl.Host + ":3100"
	}

	return parsedUrl.String(), nil
}

// odigos handles log records in otel format, e.g. with resource and log attributes.
// loki architecture works with labels, where each combination of labels values is a stream.
// This function creates processors to convert otel attributes to loki labels based on the user configuration.
func lokiLabelsProcessors(rawLabels string, exists bool, destName string) (commonconf.GenericMap, error) {

	// backwards compatibility, if the user labels are not provided, we use the default
	if !exists {
		processorName := "attributes/loki-" + destName
		return commonconf.GenericMap{
			processorName: commonconf.GenericMap{
				"actions": []commonconf.GenericMap{
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
		return commonconf.GenericMap{}, nil
	}

	var attributeNames []string
	err := json.Unmarshal([]byte(rawLabels), &attributeNames)
	if err != nil {
		return nil, err
	}
	attributeHint := strings.Join(attributeNames, ", ")

	processors := commonconf.GenericMap{}

	// since we don't know if the attributes are logs attributes or resource attributes, we will add them to both processors
	attributesProcessorName := "attributes/loki-" + destName
	processors[attributesProcessorName] = commonconf.GenericMap{
		"actions": []commonconf.GenericMap{
			{
				"key":    "loki.attribute.labels",
				"action": "insert",
				"value":  attributeHint,
			},
		},
	}

	resourceProcessorName := "resource/loki-" + destName
	processors[resourceProcessorName] = commonconf.GenericMap{
		"attributes": []commonconf.GenericMap{
			{
				"key":    "loki.resource.labels",
				"action": "insert",
				"value":  attributeHint,
			},
		},
	}

	return processors, nil
}
