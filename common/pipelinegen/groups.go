// Package pipelinegen contains helper structures for dynamically generating OpenTelemetry
// pipelines based on source groups and their associated destinations.
//
// This file (`groups.go`) defines the core data structures used to:
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

// GroupDetails defines a logical group of source workloads and the destination exporters
// that they are allowed to send observability data to.
type GroupDetails struct {
	Name         string            `mapstructure:"name"`         // Unique identifier for the group (used as pipeline name suffix)
	Namespaces   []NamespaceFilter `mapstructure:"namespaces"`   // List of namespaces belonging to this group [marked as future select]
	Sources      []SourceFilter    `mapstructure:"sources"`      // List of workloads belonging to this group
	Destinations []Destination     `mapstructure:"destinations"` // List of destination IDs this group routes data to
}

// SourceFilter represents a single K8s source workload that will emit observability data.
// It is uniquely identified by its Namespace, Kind (e.g. Deployment, StatefulSet), and Name.
type SourceFilter struct {
	Namespace string `mapstructure:"namespace"` // K8s namespace of the workload
	Kind      string `mapstructure:"kind"`      // Workload kind: Deployment, StatefulSet, etc.
	Name      string `mapstructure:"name"`      // Name of the specific workload
}

// NamespaceFilter represents a single K8s namespace that will emit observability data.
// It is uniquely identified by its name. this handled for marked as "future select"
// In that case a single source will be created for the namespace.
type NamespaceFilter struct {
	Namespace string `mapstructure:"namespace"` // K8s namespace of the namespace
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
var telemetryRootPipelinesBySignal = map[string]string{
	strings.ToLower(string(common.TracesObservabilitySignal)):  strings.ToLower(string(common.TracesObservabilitySignal)) + "/in",
	strings.ToLower(string(common.MetricsObservabilitySignal)): strings.ToLower(string(common.MetricsObservabilitySignal)) + "/in",
	strings.ToLower(string(common.LogsObservabilitySignal)):    strings.ToLower(string(common.LogsObservabilitySignal)) + "/in",
}

func GetTelemetryRootPipeline(signal string) string {
	return telemetryRootPipelinesBySignal[signal]
}
func GetSignalsRootPipelines() []string {
	return []string{
		GetTelemetryRootPipeline(strings.ToLower(string(common.TracesObservabilitySignal))),
		GetTelemetryRootPipeline(strings.ToLower(string(common.MetricsObservabilitySignal))),
		GetTelemetryRootPipeline(strings.ToLower(string(common.LogsObservabilitySignal))),
	}
}
