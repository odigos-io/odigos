package k8sconsts

const (
	NodeNameEnvVar = "NODE_NAME"
	// set on the odiglet init container when the chart runs with openshift.enabled
	OpenShiftEnabledEnvVar = "ODIGOS_OPENSHIFT_ENABLED"
	NodeIPEnvVar           = "NODE_IP"

	GKEAutopilotEnvVar = "GKE_AUTOPILOT"

	// GKEManagedNodeNamePrefix is the prefix for node names on GKE Autopilot clusters.
	GKEManagedNodeNamePrefix = "gk3-"

	// FirstInstrumentedPodAtNodeLabel is set on a Node by the instrumentor when the
	// first Odigos-instrumented pod is discovered on that node.
	// The value is the UTC time of that first discovery
	// (compact form 20060102T150405Z; colons are not valid in label values).
	// The label is removed only after no instrumented pods remain and the label is
	// at least as old as FirstInstrumentedPodAtNodeLabelRetentionEnvVar.
	// When that env var is set, odiglet DaemonSet pods are scheduled only onto
	// Nodes that carry this label (Exists match).
	FirstInstrumentedPodAtNodeLabel = "odigos.io/first-instrumented-pod-at"

	// FirstInstrumentedPodAtNodeLabelTimeFormat is the label-safe UTC timestamp layout
	// used as the value of FirstInstrumentedPodAtNodeLabel.
	FirstInstrumentedPodAtNodeLabelTimeFormat = "20060102T150405Z"

	// FirstInstrumentedPodAtNodeLabelRetentionEnvVar is a Go duration (e.g. "5m") set on
	// the instrumentor from odiglet.scheduleOnlyOnInstrumentedNodes when enabled.
	// Presence of this env var enables scheduling odiglet only on instrumented nodes
	// and registers the instrumented-nodes controllers. The value is how long to keep
	// FirstInstrumentedPodAtNodeLabel after the last instrumented pod leaves (use "0s"
	// to remove immediately).
	FirstInstrumentedPodAtNodeLabelRetentionEnvVar = "ODIGOS_INSTRUMENTED_PODS_NODE_LABEL_RETENTION"
)
