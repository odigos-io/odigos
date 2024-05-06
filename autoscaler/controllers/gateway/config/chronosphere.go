package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
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

func (c *Chronosphere) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) error {

	url, exists := dest.Spec.Data[chronosphereDomain]
	if !exists {
		return ErrorChronosphereMissingURL
	}

	company := c.getCompanyNameFromURL(url)

	chronosphereExporterName := "otlp/chronosphere-" + dest.Name
	currentConfig.Exporters[chronosphereExporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s.chronosphere.io:443", company),
		"retry_on_failure": commonconf.GenericMap{
			"enabled": true,
		},
		"compression": "gzip",
		"headers": commonconf.GenericMap{
			"API-Token": "${CHRONOSPHERE_API_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		tracePipelineName := "traces/chronosphere-" + dest.Name
		currentConfig.Service.Pipelines[tracePipelineName] = commonconf.Pipeline{
			Exporters: []string{chronosphereExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		// Set service.instance.id to pod name or node name
		chronosphereMetricProcessorName := "resource/chornosphere-" + dest.Name
		currentConfig.Processors[chronosphereMetricProcessorName] = commonconf.GenericMap{
			"attributes": []commonconf.GenericMap{
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

		metricsPipelineName := "metrics/chronosphere-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters:  []string{chronosphereExporterName},
			Processors: []string{chronosphereMetricProcessorName},
		}
	}

	return nil
}

func (c *Chronosphere) getCompanyNameFromURL(url string) string {
	// Remove trailing slash if present
	url = strings.TrimSuffix(url, "/")

	// Support the following cases: COMAPNY / COMPANY.chronosphere.io / COMPANY.chronosphere.io:443
	url = strings.TrimSuffix(url, ".chronosphere.io:443")
	url = strings.TrimSuffix(url, ".chronosphere.io")
	return url
}
