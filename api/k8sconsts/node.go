package k8sconsts

const (
	NodeNameEnvVar = "NODE_NAME"
	NodeIPEnvVar   = "NODE_IP"

	GKEAutopilotEnvVar = "GKE_AUTOPILOT"

	// GKEManagedNodeNamePrefix is the prefix for node names on GKE Autopilot clusters.
	GKEManagedNodeNamePrefix = "gk3-"

	// FirstInstrumentedPodAtNodeLabel is set on a Node by the instrumentor when the
	// first Odigos-instrumented pod is discovered on that node.
	// The value is the UTC time of that first discovery
	// (compact form 20060102T150405Z; colons are not valid in label values).
	// The label is removed only after no instrumented pods remain and the label is
	// at least FirstInstrumentedPodAtNodeLabelRetentionEnvVar old.
	// When odiglet.scheduleOnlyOnInstrumentedNodes.enabled is true, odiglet DaemonSet
	// pods are scheduled only onto Nodes that carry this label (Exists match).
	FirstInstrumentedPodAtNodeLabel = "odigos.io/first-instrumented-pod-at"

	// FirstInstrumentedPodAtNodeLabelTimeFormat is the label-safe UTC timestamp layout
	// used as the value of FirstInstrumentedPodAtNodeLabel.
	FirstInstrumentedPodAtNodeLabelTimeFormat = "20060102T150405Z"

	// OdigletScheduleOnlyOnInstrumentedNodesEnvVar is set on the instrumentor from
	// odiglet.scheduleOnlyOnInstrumentedNodes.enabled. Changing it rolls the instrumentor
	// and controls whether the instrumented-nodes controllers are registered.
	OdigletScheduleOnlyOnInstrumentedNodesEnvVar = "ODIGOS_ODIGLET_SCHEDULE_ONLY_ON_INSTRUMENTED_NODES"

	// FirstInstrumentedPodAtNodeLabelRetentionEnvVar is a Go duration (e.g. "5m") set on the
	// instrumentor from odiglet.scheduleOnlyOnInstrumentedNodes.nodeLabelRetention.
	// FirstInstrumentedPodAtNodeLabel is not removed until it has existed at least this long.
	FirstInstrumentedPodAtNodeLabelRetentionEnvVar = "ODIGOS_INSTRUMENTED_PODS_NODE_LABEL_RETENTION"
)
