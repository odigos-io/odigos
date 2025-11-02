// Package pipelinegen contains helper structures for dynamically generating OpenTelemetry
// pipelines based on source groups and their associated destinations.
//
// This file (`datastreams.go`) defines the core data structures used to:
// - Describe source filters (which K8s workloads send data)
// - Associate groups of sources with destination exporters
// - Track exporter metadata (like supported signals)
//
// These structures are consumed by routing and pipeline generation logic.

package pipelinegen

import (
	"strings"

	"github.com/odigos-io/odigos/common"
)

// DataStreams defines a logical group of source workloads and the destination exporters
// that they are allowed to send observability data to.
type DataStreams struct {
	Name         string        `mapstructure:"name"`         // Unique identifier for the group (used as pipeline name suffix)
	Destinations []Destination `mapstructure:"destinations"` // List of destination IDs this group routes data to
}

// Destination represents a destination that a group can send data to.
// It includes the destination name and the signals the user configured to be sent to this destination.
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
