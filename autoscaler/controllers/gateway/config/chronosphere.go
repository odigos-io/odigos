package config

import (
	"errors"
	"fmt"
	"strings"

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	commonconf "github.com/keyval-dev/odigos/autoscaler/controllers/common"
	"github.com/keyval-dev/odigos/common"
	ctrl "sigs.k8s.io/controller-runtime"
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

func (c *Chronosphere) ModifyConfig(dest *odigosv1.Destination, currentConfig *commonconf.Config) {

	url, exists := dest.Spec.Data[chronosphereDomain]
	if !exists {
		ctrl.Log.Error(ErrorChronosphereMissingURL, "skipping Chronosphere destination config")
		return
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
		metricsPipelineName := "metrics/chronosphere-" + dest.Name
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{chronosphereExporterName},
		}
	}
}

func (c *Chronosphere) getCompanyNameFromURL(url string) string {
	// Remove trailing slash if present
	url = strings.TrimSuffix(url, "/")

	// Support the following cases: COMAPNY / COMPANY.chronosphere.io / COMPANY.chronosphere.io:443
	url = strings.TrimSuffix(url, ".chronosphere.io:443")
	url = strings.TrimSuffix(url, ".chronosphere.io")
	return url
}
