package config

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/odigos-io/odigos/common"
)

const (
	logScaleEndpointKey    = "LOGSCALE_ENDPOINT"
	logScaleIngestTokenKey = "LOGSCALE_INGEST_TOKEN"
)

type LogScale struct{}

func (l *LogScale) DestType() common.DestinationType {
	return common.LogScaleDestinationType
}

func (l *LogScale) ModifyConfig(dest ExporterConfigurer, cfg *Config) ([]string, error) {
	for _, signal := range dest.GetSignals() {
		if signal != common.LogsObservabilitySignal {
			return nil, fmt.Errorf("Falcon LogScale supports only logs in Odigos; remove the %s signal", signal)
		}
	}
	rawEndpoint := strings.TrimSpace(dest.GetConfig()[logScaleEndpointKey])
	if rawEndpoint == "" {
		return nil, errorMissingKey(logScaleEndpointKey)
	}
	endpoint, err := parseOtlpHttpEndpoint(strings.TrimRight(rawEndpoint, "/"), "", "/api/v1/ingest/otlp")
	if err != nil {
		return nil, fmt.Errorf("invalid %s: provide the LogScale instance URL or /api/v1/ingest/otlp base endpoint: %w", logScaleEndpointKey, err)
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Hostname() == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, fmt.Errorf("invalid %s: a host is required; credentials, query parameters, and fragments are not allowed", logScaleEndpointKey)
	}
	if !isLoggingEnabled(dest) {
		return nil, nil
	}
	uniqueURI := "logscale-" + dest.GetID()
	exporterName := "otlp_http/" + uniqueURI
	cfg.Exporters[exporterName] = GenericMap{
		"endpoint":    endpoint,
		"encoding":    "proto",
		"compression": "none",
		"sending_queue": GenericMap{
			"batch": GenericMap{
				"sizer":         "items",
				"min_size":      1024,
				"max_size":      5000,
				"flush_timeout": "5s",
			},
		},
		"headers": GenericMap{
			"Authorization": "Bearer " + SecretEnvPlaceholder(logScaleIngestTokenKey, dest),
		},
	}
	pipelineName := "logs/" + uniqueURI
	cfg.Service.Pipelines[pipelineName] = Pipeline{Exporters: []string{exporterName}}
	return []string{pipelineName}, nil
}
