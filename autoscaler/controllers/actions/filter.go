package actions

import (
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
)

// regexMetaChars contains characters that indicate a regex pattern
const regexMetaChars = `^$*+?{[]()|\`

// isRegexPattern checks if a string contains regex metacharacters
func isRegexPattern(s string) bool {
	return strings.ContainsAny(s, regexMetaChars)
}

type filterProcessorConfig struct {
	ErrorMode string        `json:"error_mode"`
	Traces    tracesConfig  `json:"traces"`
	Metrics   metricsConfig `json:"metrics"`
	Logs      logsConfig    `json:"logs"`
}

type tracesConfig struct {
	Span      []string `json:"span"`
	SpanEvent []string `json:"spanevent"`
}

type metricsConfig struct {
	Metric    []string `json:"metric"`
	DataPoint []string `json:"datapoint"`
}

type logsConfig struct {
	LogRecord []string `json:"log_record"`
}

func attributeBasedFiltersConfig(attributes map[string]string, signals []common.ObservabilitySignal) (any, error) {
	config := filterProcessorConfig{
		ErrorMode: "ignore",
	}

	// Build filter lists once
	var filters []string
	for key, value := range attributes {
		if isRegexPattern(value) {
			filters = append(filters, fmt.Sprintf("IsMatch(attributes[\"%s\"], \"%s\")", key, value))
			filters = append(filters, fmt.Sprintf("IsMatch(resource.attributes[\"%s\"], \"%s\")", key, value))
		} else {
			filters = append(filters, fmt.Sprintf("attributes[\"%s\"] == \"%s\"", key, value))
			filters = append(filters, fmt.Sprintf("resource.attributes[\"%s\"] == \"%s\"", key, value))
		}
	}

	// Apply to each enabled signal
	for _, signal := range signals {
		switch signal {
		case common.TracesObservabilitySignal:
			config.Traces = tracesConfig{
				Span: filters,
			}
		case common.MetricsObservabilitySignal:
			config.Metrics = metricsConfig{
				Metric:    filters,
				DataPoint: filters,
			}
		case common.LogsObservabilitySignal:
			config.Logs = logsConfig{
				LogRecord: filters,
			}
		}
	}

	return config, nil
}
