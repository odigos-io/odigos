package config

import (
	"errors"
	"fmt"
	"strings"

	commonconf "github.com/odigos-io/odigos/autoscaler/controllers/common"
	"github.com/odigos-io/odigos/common"
)

const (
	signozUrlKey = "SIGNOZ_URL"
)

type Signoz struct{}

func (s *Signoz) DestType() common.DestinationType {
	return common.SignozDestinationType
}

func (s *Signoz) ModifyConfig(dest common.ExporterConfigurer, currentConfig *commonconf.Config) error {
	url, exists := dest.GetConfig()[signozUrlKey]
	if !exists {
		return errors.New("Signoz url not specified, gateway will not be configured for Signoz")
	}

	if strings.HasPrefix(url, "https://") {
		return errors.New("Signoz does not currently supports tls export, gateway will not be configured for Signoz")
	}

	url = strings.TrimPrefix(url, "http://")
	url = strings.TrimSuffix(url, ":4317")
	signozExporterName := "otlp/signoz-" + dest.GetName()
	currentConfig.Exporters[signozExporterName] = commonconf.GenericMap{
		"endpoint": fmt.Sprintf("%s:4317", url),
		"tls": commonconf.GenericMap{
			"insecure": true,
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/signoz-" + dest.GetName()
		currentConfig.Service.Pipelines[tracesPipelineName] = commonconf.Pipeline{
			Exporters: []string{signozExporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/signoz-" + dest.GetName()
		currentConfig.Service.Pipelines[metricsPipelineName] = commonconf.Pipeline{
			Exporters: []string{signozExporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/signoz-" + dest.GetName()
		currentConfig.Service.Pipelines[logsPipelineName] = commonconf.Pipeline{
			Exporters: []string{signozExporterName},
		}
	}

	return nil
}
