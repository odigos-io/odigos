package consts

type CollectorType string

const (
	// Cluster collector is responsible for exporting observability data from the cluster.
	ClusterCollector CollectorType = "cluster"
	// Node collector is receiving data from different instrumentation SDKs in the same node.
	NodeCollector    CollectorType = "node"
)

// OdigosCollectorRoleLabel is the label used to identify the role of the Odigos collector.
const OdigosCollectorRoleLabel = "odigos.io/collector-role"

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
