package common

import "github.com/odigos-io/odigos/api/k8sconsts"

type SourceID struct {
	// combination of namespace, kind and name is unique
	Name      string                 `json:"name"`
	Kind      k8sconsts.WorkloadKind `json:"kind"`
	Namespace string                 `json:"namespace"`
}
