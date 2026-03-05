package api

import (
	"github.com/odigos-io/odigos/common"
)

// WorkloadRef identifies a Kubernetes workload (namespace, kind, name).
// Used by SourcesScope and scope matching; kept in common to avoid common depending on api/k8sconsts.
type WorkloadRef struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Kind      string `json:"kind"` // e.g. "Deployment", "StatefulSet"
}

// define conditions to match specific sources (containers) managed by odigos.
// a source container matches, if ALL non empty fields match (AND semantics)
//
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
	WorkloadName      string `json:"workloadName,omitempty"`
	WorkloadKind      string `json:"workloadKind,omitempty"` // e.g. "Deployment"
	WorkloadNamespace string `json:"workloadNamespace,omitempty"`
	ContainerName     string `json:"containerName,omitempty"`

	WorkloadLanguage common.ProgrammingLanguage `json:"workloadLanguage,omitempty"`
}
