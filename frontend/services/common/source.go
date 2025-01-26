package common

import "github.com/odigos-io/odigos/k8sutils/pkg/workload"

type SourceID struct {
	// combination of namespace, kind and name is unique
	Name      string                `json:"name"`
	Kind      workload.WorkloadKind `json:"kind"`
	Namespace string                `json:"namespace"`
}
