package consts

const (
	OdigosClusterCollectorDeploymentName     = "odigos-gateway"
	OdigosClusterCollectorConfigMapName      = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorServiceName        = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorCollectorGroupName = OdigosClusterCollectorDeploymentName

	OdigosNodeCollectorDaemonSetName      = "odigos-data-collection"
	OdigosNodeCollectorConfigMapName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName = OdigosNodeCollectorDaemonSetName
)
