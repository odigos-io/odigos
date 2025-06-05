package k8sconsts

type CollectorRole string

const (
	CollectorsRoleClusterGateway CollectorRole = "CLUSTER_GATEWAY"
	CollectorsRoleNodeCollector  CollectorRole = "NODE_COLLECTOR"
)

const OdigosCollectorConfigMapProviderScheme = "k8scm"
