package consts

type CollectorRole string

const (
	CollectorsRoleClusterGateway CollectorRole = "CLUSTER_GATEWAY"
	CollectorsRoleNodeCollector  CollectorRole = "NODE_COLLECTOR"
)

// OdigosCollectorRoleLabel is the label used to identify the role of the Odigos collector.
const OdigosCollectorRoleLabel = "odigos.io/collector-role"

const (
	OdigosDeploymentConfigMapName = "odigos-deployment"
)

const (
	OdigosClusterCollectorDeploymentName = "odigos-gateway"
	OdigosClusterCollectorConfigMapName  = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorServiceName    = OdigosClusterCollectorDeploymentName

	OdigosClusterCollectorCollectorGroupName = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorConfigMapKey       = "collector-conf"
)

const (
	OdigosNodeCollectorDaemonSetName      = "odigos-data-collection"
	OdigosNodeCollectorConfigMapName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName = OdigosNodeCollectorDaemonSetName

	OdigosNodeCollectorConfigMapKey = "conf" // this key is different than the cluster collector value. not sure why
)
