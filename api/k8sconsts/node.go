package k8sconsts

const (
	NodeNameEnvVar = "NODE_NAME"
	NodeIPEnvVar   = "NODE_IP"

	GKEAutopilotEnvVar = "GKE_AUTOPILOT"

	// GKEManagedNodeNamePrefix is the prefix for node names on GKE Autopilot clusters.
	GKEManagedNodeNamePrefix = "gk3-"

	// InstrumentedPodsNodeLabel is set on a Node by the instrumentor when at least
	// one Odigos-instrumented pod is running on that node.
	// When odiglet.scheduleOnlyOnInstrumentedNodes is enabled, odiglet DaemonSet
	// pods are scheduled only onto Nodes that carry this label.
	InstrumentedPodsNodeLabel = "odigos.io/instrumented-pods"

	// InstrumentedPodsNodeLabelValue is the value written for InstrumentedPodsNodeLabel
	// when instrumented pods are present on the node. The label is removed when none remain.
	InstrumentedPodsNodeLabelValue = "true"

	// OdigletScheduleOnlyOnInstrumentedNodesEnvVar is set on the instrumentor from
	// odiglet.scheduleOnlyOnInstrumentedNodes. Changing it rolls the instrumentor and
	// controls whether the instrumented-nodes controllers are registered.
	OdigletScheduleOnlyOnInstrumentedNodesEnvVar = "ODIGOS_ODIGLET_SCHEDULE_ONLY_ON_INSTRUMENTED_NODES"
)
