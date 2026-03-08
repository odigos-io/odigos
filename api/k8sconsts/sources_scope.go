package k8sconsts

import "github.com/odigos-io/odigos/common"

// define conditions to match specific sources (containers) managed by odigos.
// a source container matches, if ALL non empty fields match (AND semantics)
// common patterns:
//   - Specific kubernetes workload by name (WorkloadNamespace + WorkloadKind + WorkloadName):
//     all containers (usually there is only one with agent injection)
//   - Specific container in a kubernetes workload (WorkloadNamespace + WorkloadKind + WorkloadName + ContainerName):
//     only this container
//   - All services in a kubernetes namespace (WorkloadNamespace):
//     all containers in all sources in the namespace
//   - All services implemented in a specific programming language (WorkloadLanguage):
//     all container which are running odigos agent for this language
//
// +kubebuilder:object:generate=true
// +kubebuilder:deepcopy-gen=true
type SourcesScope struct {
	WorkloadName      string                     `json:"workloadName,omitempty"`
	WorkloadKind      string                     `json:"workloadKind,omitempty"` // e.g. "Deployment"
	WorkloadNamespace string                     `json:"workloadNamespace,omitempty"`
	ContainerName     string                     `json:"containerName,omitempty"`
	WorkloadLanguage  common.ProgrammingLanguage `json:"workloadLanguage,omitempty"`
}
