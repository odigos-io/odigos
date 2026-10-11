package loaders

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/frontend/graph/model"
)

type WorkloadFilterSingleWorkload struct {
	WorkloadKind k8sconsts.WorkloadKind
	WorkloadName string
	Namespace    string
}

type WorkloadFilterSingleNamespace struct {
	Namespace string
}

type WorkloadFilterClusterWide struct {
}

type WorkloadFilter struct {
	SingleWorkload  *WorkloadFilterSingleWorkload
	SingleNamespace *WorkloadFilterSingleNamespace
	ClusterWide     *WorkloadFilterClusterWide

	// set to relevant namespace if applicable, or empty string if cluster wide.
	// can be used in k8s client calls to populate the namespace field.
	NamespaceString string

	// namespaces to ignore when fetching instrumentation configs.
	// if the namespace name is in this map, it will be ignored when fetching k8s resources.
	IgnoredNamespaces map[string]struct{}

	// workload IDs that bypass the ignored-namespace filter. Populated from the cluster-wide
	// InstrumentationConfig list — a workload that has an IC was explicitly instrumented (typically
	// via a manifest-created Source CR in an ignored namespace like odigos-system), so the user
	// should see and be able to mutate it even though its namespace is ignored elsewhere in the UI.
	BypassWorkloads map[model.K8sWorkloadID]struct{}
}

// ShouldIgnoreWorkload reports whether a workload should be excluded due to its namespace being ignored.
// Workloads in ignored namespaces are excluded unless they appear in BypassWorkloads (i.e. they have an
// InstrumentationConfig and were explicitly instrumented by the user).
func (f *WorkloadFilter) ShouldIgnoreWorkload(id model.K8sWorkloadID) bool {
	if f == nil {
		return false
	}
	if _, ignored := f.IgnoredNamespaces[id.Namespace]; !ignored {
		return false
	}
	if _, bypass := f.BypassWorkloads[id]; bypass {
		return false
	}
	return true
}
