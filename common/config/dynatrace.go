package config

import (
	"errors"
	"fmt"
	"net/url"

	"github.com/odigos-io/odigos/common"
)

const (
	dynatraceURLKey      = "DYNATRACE_URL"
	dynatraceAPITOKENKey = "${DYNATRACE_API_TOKEN}"
)

var (
	ErrDynatraceURLNotSpecified      = fmt.Errorf("Dynatrace url  not specified")
	ErrDynatraceAPITOKENNotSpecified = fmt.Errorf("Api token not specified")
)

type Dynatrace struct{}

func (n *Dynatrace) DestType() common.DestinationType {
	return common.DynatraceDestinationType
}

func (n *Dynatrace) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) error {
	if !n.requiredVarsExists(dest) {
		return errors.New("Dynatrace config is missing required variables")
	}

	baseURL, err := parsetheDTurl(dest.GetConfig()[dynatraceURLKey])
	if err != nil {
		return errors.New("Dynatrace url is not a valid")
	}

	exporterName := "otlphttp/dynatrace-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": baseURL + "/api/v2/otlp",
		"headers": GenericMap{
			"Authorization": "Api-Token ${DYNATRACE_API_TOKEN}",
		},
	}

	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/dynatrace-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/dynatrace-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/dynatrace-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
	}

	return nil
}

func (g *Dynatrace) requiredVarsExists(dest ExporterConfigurer) bool {
	if _, ok := dest.GetConfig()[dynatraceURLKey]; !ok {
		return false
	}
	return true
}

func parsetheDTurl(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		return parsetheDTurl(fmt.Sprintf("https://%s", rawURL))
	}

	return fmt.Sprintf("https://%s", u.Host), nil
}
