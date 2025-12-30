package k8sconsts

import "k8s.io/apimachinery/pkg/runtime/schema"

// GroupVersionKind constants for external workload types.
// These can be used to check resource availability in clusters.
var (
	DeploymentConfigGVK = schema.GroupVersionKind{
		Group:   "apps.openshift.io",
		Version: "v1",
		Kind:    string(WorkloadKindDeploymentConfig),
	}
	ArgoRolloutGVK = schema.GroupVersionKind{
		Group:   "argoproj.io",
		Version: "v1alpha1",
		Kind:    string(WorkloadKindArgoRollout),
	}
)

// 1. the pascal case representation of the workload kind
// it is used in k8s api objects as the `Kind` field.
type WorkloadKind string

const (
	WorkloadKindDeployment       WorkloadKind = "Deployment"
	WorkloadKindDaemonSet        WorkloadKind = "DaemonSet"
	WorkloadKindStatefulSet      WorkloadKind = "StatefulSet"
	WorkloadKindNamespace        WorkloadKind = "Namespace"
	WorkloadKindCronJob          WorkloadKind = "CronJob"
	WorkloadKindJob              WorkloadKind = "Job"
	WorkloadKindDeploymentConfig WorkloadKind = "DeploymentConfig"
	// Note: The actual Kubernetes resource has kind "Rollout" (not "ArgoRollout"):
	//   apiVersion: argoproj.io/v1alpha1
	//   kind: Rollout
	// We use "ArgoRollout" in the variable name to distinguish it from other rollout concepts.
	WorkloadKindArgoRollout WorkloadKind = "Rollout"
)

// 2. the lower case representation of the workload kind
// is used in odigos with the object name for instrumentation config and runtime details
type WorkloadKindLowerCase string

const (
	WorkloadKindLowerCaseDeployment       WorkloadKindLowerCase = "deployment"
	WorkloadKindLowerCaseDaemonSet        WorkloadKindLowerCase = "daemonset"
	WorkloadKindLowerCaseStatefulSet      WorkloadKindLowerCase = "statefulset"
	WorkloadKindLowerCaseNamespace        WorkloadKindLowerCase = "namespace"
	WorkloadKindLowerCaseCronJob          WorkloadKindLowerCase = "cronjob"
	WorkloadKindLowerCaseJob              WorkloadKindLowerCase = "job"
	WorkloadKindLowerCaseDeploymentConfig WorkloadKindLowerCase = "deploymentconfig"
	WorkloadKindLowerCaseArgoRollout      WorkloadKindLowerCase = "rollout"
)

// PodWorkload represents the higher-level controller managing a specific Pod within a Kubernetes cluster.
// It contains essential details about the controller such as its Name, Namespace, and Kind.
// 'Kind' refers to the type of controller, which can be a Deployment, StatefulSet, or DaemonSet.
// This struct is useful for identifying and interacting with the overarching entity
// that governs the lifecycle and behavior of a Pod, especially in contexts where
// understanding the relationship between a Pod and its controlling workload is crucial.
type PodWorkload struct {
	Name      string       `json:"name"`
	Namespace string       `json:"namespace"`
	Kind      WorkloadKind `json:"kind"`
}
