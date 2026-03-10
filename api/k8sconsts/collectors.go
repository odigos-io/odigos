package k8sconsts

type CollectorRole string

const (
	CollectorsRoleClusterGateway CollectorRole = "CLUSTER_GATEWAY"
	CollectorsRoleNodeCollector  CollectorRole = "NODE_COLLECTOR"
)

const OdigosCollectorConfigMapProviderScheme = "k8scm"

const OdigosConfigK8sExtensionType = "odigos_config_k8s"

// URL templatization: synthetic reconcile key and Action label for server-side filtering.
// Used by the autoscaler action controller (Processor watcher and listActionsWithUrlTemplatization).
const (
	// URLTemplatizationNamespaceSyncKey is a synthetic reconcile key; invalid as a k8s object name to avoid collisions with real Actions.
	URLTemplatizationNamespaceSyncKey = "__odigos_url_templatization_ns_sync__"
	// URLTemplatizationLabelKey is the label set on Actions that have URLTemplatization and are not disabled.
	URLTemplatizationLabelKey   = "odigos.io/url-templatization"
	URLTemplatizationLabelValue = "true"
)
