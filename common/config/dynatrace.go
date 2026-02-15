package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	dynatraceURLKey = "DYNATRACE_URL"
)

var (
	ErrDynatraceURLNotSpecified      = fmt.Errorf("Dynatrace url  not specified")
	ErrDynatraceAPITOKENNotSpecified = fmt.Errorf("api token not specified")
)

type Dynatrace struct{}

func (n *Dynatrace) DestType() common.DestinationType {
	return common.DynatraceDestinationType
}

func (n *Dynatrace) ModifyConfig(dest ExporterConfigurer, currentConfig *Config) ([]string, error) {
	if !n.requiredVarsExists(dest) {
		return nil, errors.New("Dynatrace config is missing required variables")
	}

	baseURL, err := parsetheDTurl(dest.GetConfig()[dynatraceURLKey])
	if err != nil {
		return nil, errors.New("Dynatrace url is not a valid")
	}

	exporterName := "otlphttp/dynatrace-" + dest.GetID()
	currentConfig.Exporters[exporterName] = GenericMap{
		"endpoint": baseURL + "/api/v2/otlp",
		"headers": GenericMap{
			"Authorization": "Api-Token ${DYNATRACE_API_TOKEN}",
		},
	}
	var pipelineNames []string
	if isTracingEnabled(dest) {
		tracesPipelineName := "traces/dynatrace-" + dest.GetID()
		currentConfig.Service.Pipelines[tracesPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, tracesPipelineName)
	}

	if isMetricsEnabled(dest) {
		metricsPipelineName := "metrics/dynatrace-" + dest.GetID()
		currentConfig.Service.Pipelines[metricsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, metricsPipelineName)
	}

	if isLoggingEnabled(dest) {
		logsPipelineName := "logs/dynatrace-" + dest.GetID()
		currentConfig.Service.Pipelines[logsPipelineName] = Pipeline{
			Exporters: []string{exporterName},
		}
		pipelineNames = append(pipelineNames, logsPipelineName)
	}

	return pipelineNames, nil
}

func (n *Dynatrace) requiredVarsExists(dest ExporterConfigurer) bool {
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

	// Preserve the URL path for Dynatrace Managed / ActiveGate environments
	// where the URL includes a path component like /e/{environment-id}
	path := strings.TrimRight(u.Path, "/")
	return fmt.Sprintf("https://%s%s", u.Host, path), nil
}
