package k8sconsts

const (
	// These are currently "magic" numbers that we are using to set the QPS and Burst for the Kubernetes client.
	// They allow for better performance relative to the default values, but with the cost of potentially
	// overloading the Kubernetes API server.
	// More info about these can be found in https://kubernetes.io/docs/reference/config-api/apiserver-eventratelimit.v1alpha1/
	K8sClientDefaultQPS   = 100
	K8sClientDefaultBurst = 100
)
