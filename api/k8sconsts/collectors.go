package k8sconsts

type CollectorRole string

const (
	CollectorsRoleClusterGateway CollectorRole = "CLUSTER_GATEWAY"
	CollectorsRoleNodeCollector  CollectorRole = "NODE_COLLECTOR"
)

const OdigosCollectorConfigMapProviderScheme = "k8scm"

const OdigosConfigK8sExtensionType = "odigos_config_k8s"
