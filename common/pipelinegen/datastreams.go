package pipelinegen

import (
	"strings"

	"github.com/odigos-io/odigos/common"
)

// DataStreams defines a logical grouping of destination exporters under a named pipeline.
// Source-to-datastream membership is resolved dynamically via the odigosconfigk8sextension.
type DataStreams struct {
	Name         string        `mapstructure:"name"`
	Destinations []Destination `mapstructure:"destinations"`
}

// Destination represents a destination that a data stream can send data to.
type Destination struct {
	DestinationName   string                       `mapstructure:"destinationname"`
	ConfiguredSignals []common.ObservabilitySignal `mapstructure:"configuredsignals"`
}

// telemetryRootPipelinesBySignal maps observability signal types to their corresponding
// root pipelines. This mapping helps identify the initial pipeline for a given signal
// when building the telemetry configuration.
// and also to be single source of truth for the root pipelines
var telemetryRootPipelinesBySignal = map[common.ObservabilitySignal]string{
	common.TracesObservabilitySignal:  strings.ToLower(string(common.TracesObservabilitySignal)) + "/in",
	common.MetricsObservabilitySignal: strings.ToLower(string(common.MetricsObservabilitySignal)) + "/in",
	common.LogsObservabilitySignal:    strings.ToLower(string(common.LogsObservabilitySignal)) + "/in",
}

func GetTelemetryRootPipelineName(signal common.ObservabilitySignal) string {
	return telemetryRootPipelinesBySignal[signal]
}
func GetSignalsRootPipelineNames() []string {
	return []string{
		GetTelemetryRootPipelineName(common.TracesObservabilitySignal),
		GetTelemetryRootPipelineName(common.MetricsObservabilitySignal),
		GetTelemetryRootPipelineName(common.LogsObservabilitySignal),
	}
}
