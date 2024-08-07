package consts

const (
	OdigosClusterCollectorDeploymentName     = "odigos-gateway"
	OdigosClusterCollectorConfigMapName      = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorServiceName        = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorCollectorGroupName = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorConfigMapKey       = "collector-conf"

	OdigosNodeCollectorDaemonSetName      = "odigos-data-collection"
	OdigosNodeCollectorConfigMapName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorConfigMapKey       = "conf" // this key is different than the cluster collector value. not sure why
)
