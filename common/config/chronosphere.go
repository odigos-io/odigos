package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

var (
	ErrorChronosphereMissingURL = errors.New("missing CHRONOSPHERE_DOMAIN config")
)

const (
	chronosphereDomain = "CHRONOSPHERE_DOMAIN"
)

type Chronosphere struct{}

func (c *Chronosphere) DestType() common.DestinationType {
	return common.ChronosphereDestinationType
}

func (c *Chronosphere) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	url, exists := dest.GetConfig()[chronosphereDomain]
	if !exists {
		return nil, ErrorChronosphereMissingURL
	}

	company := c.getCompanyNameFromURL(url)

	chronosphereExporterName := "otlp/chronosphere-" + dest.GetID()
	currentConfig.Exporters[chronosphereExporterName] = GenericMap{
		"endpoint": fmt.Sprintf("%s.chronosphere.io:443", company),
		"retry_on_failure": GenericMap{
			"enabled": true,
		},
		"compression": "gzip",
		"headers": GenericMap{
			"API-Token": "${CHRONOSPHERE_API_TOKEN}",
		},
	}
	var pipelineNames []string

	if isTracingEnabled(dest) {
		tracePipelineName := "traces/chronosphere-" + dest.GetID()
		currentConfig.Service.Pipelines[tracePipelineName] = Pipeline{
			Exporters: []string{chronosphereExporterName},
		}
		pipelineNames = append(pipelineNames, tracePipelineName)
	}

	if isMetricsEnabled(dest) {
		// Set service.instance.id to pod name or node name
		chronosphereMetricProcessorName := "resource/chornosphere-" + dest.GetID()
		currentConfig.Processors[chronosphereMetricProcessorName] = GenericMap{
			"attributes": []GenericMap{
				{
					"key":            "service.instance.id",
					"from_attribute": "k8s.node.name",
					"action":         "insert",
				},
				{
					"key":            "service.instance.id",
					"from_attribute": "k8s.pod.name",
					"action":         "upsert",
				},
				{
					"key":    "instrumentation.control.plane",
					"value":  "odigos",
					"action": "insert",
				},
			},
		}

		metricsPipelineName := "metrics/chronosphere-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters:  []string{chronosphereExporterName},
			Processors: []string{chronosphereMetricProcessorName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	return pipelineNames, nil
}

func (c *Chronosphere) getCompanyNameFromURL(url string) string {
	// Remove trailing slash if present
	url = strings.TrimSuffix(url, "/")

	// Support the following cases: COMPANY / COMPANY.chronosphere.io / COMPANY.chronosphere.io:443
	url = strings.TrimSuffix(url, ".chronosphere.io:443")
	url = strings.TrimSuffix(url, ".chronosphere.io")
	return url
}
