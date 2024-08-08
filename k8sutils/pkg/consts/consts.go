package consts

const (
	OdigosClusterCollectorDeploymentName = "odigos-gateway"
	OdigosClusterCollectorConfigMapName  = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorServiceName    = OdigosClusterCollectorDeploymentName
	OdigosNodeCollectorDaemonSetName     = "odigos-data-collection"

	// Label used to identify the Odigos pod which is acting as a node collector.
	OdigosNodeCollectorLabel  = "odigos.io/data-collection"
	// Label used to identify the Odigos pod which is acting as a cluster collector.
	OdigosClusterCollectorLabel = "odigos.io/gateway"
	OdigosClusterCollectorCollectorGroupName = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorConfigMapKey       = "collector-conf"

	OdigosNodeCollectorConfigMapName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorConfigMapKey       = "conf" // this key is different than the cluster collector value. not sure why
)
