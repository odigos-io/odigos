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

// SourceFilter represents a single K8s source workload that will emit observability data.
// It is uniquely identified by its Namespace, Kind (e.g. Deployment, StatefulSet), and Name.

type SourceFilter struct {
	Namespace string `mapstructure:"namespace"` // K8s namespace of the workload
	Kind      string `mapstructure:"kind"`      // Workload kind: Deployment, StatefulSet, etc.
	Name      string `mapstructure:"name"`      // Name of the specific workload
}

// GroupDetails defines a logical group of source workloads and the destination exporters
// that they are allowed to send observability data to.
type GroupDetails struct {
	Name         string         `mapstructure:"name"`         // Unique identifier for the group (used as pipeline name suffix)
	Sources      []SourceFilter `mapstructure:"sources"`      // List of workloads belonging to this group
	Destinations []string       `mapstructure:"destinations"` // List of destination IDs this group routes data to
}

// telemetryRootPipelinesBySignal maps observability signal types to their corresponding
// root pipelines. This mapping helps identify the initial pipeline for a given signal
// when building the telemetry configuration.
// and also to be single source of truth for the root pipelines
var telemetryRootPipelinesBySignal = map[string]string{
	"traces":  "traces/in",
	"metrics": "metrics/in",
	"logs":    "logs/in",
}

func GetTelemetryRootPipeline(signal string) string {
	return telemetryRootPipelinesBySignal[signal]
}
func GetSignalsRootPipelines() []string {
	return []string{
		GetTelemetryRootPipeline("traces"),
		GetTelemetryRootPipeline("metrics"),
		GetTelemetryRootPipeline("logs"),
	}
}
