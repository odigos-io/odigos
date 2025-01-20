package k8sconsts

// this should be merged with CollectorsGroupRole in collectorsgroup_types.go
type CollectorRole string

const (
	CollectorsRoleClusterGateway CollectorRole = "CLUSTER_GATEWAY"
	CollectorsRoleNodeCollector  CollectorRole = "NODE_COLLECTOR"
)
