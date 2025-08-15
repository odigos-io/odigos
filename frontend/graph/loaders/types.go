package loaders

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
}

type WorkloadManifest struct {
	AvailableReplicas int32
	Selector          *metav1.LabelSelector
	PodTemplateSpec   *corev1.PodTemplateSpec
}
