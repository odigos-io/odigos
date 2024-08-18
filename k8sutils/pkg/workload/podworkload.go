package workload

// PodWorkload represents the higher-level controller managing a specific Pod within a Kubernetes cluster.
// It contains essential details about the controller such as its Name, Namespace, and Kind.
// 'Kind' refers to the type of controller, which can be a Deployment, StatefulSet, or DaemonSet.
// This struct is useful for identifying and interacting with the overarching entity
// that governs the lifecycle and behavior of a Pod, especially in contexts where
// understanding the relationship between a Pod and its controlling workload is crucial.
type PodWorkload struct {
	Name      string
	Namespace string
	Kind      WorkloadKind
}
