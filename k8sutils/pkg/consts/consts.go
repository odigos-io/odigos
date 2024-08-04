package consts

const (
	OdigosClusterCollectorDeploymentName = "odigos-gateway"
	OdigosClusterCollectorConfigMapName  = OdigosClusterCollectorDeploymentName
	OdigosClusterCollectorServiceName    = OdigosClusterCollectorDeploymentName
	OdigosNodeCollectorDaemonSetName     = "odigos-data-collection"

	// Header key used to pass the pod name of a reporting odigos pod.
	// Used to identify the Odigos pod that is sending the data.
	OdigosPodNameHeaderKey = "odigos-pod-name"
	// Label used to identify the Odigos pod which is acting as a node collector.
	OdigosNodeCollectorLabel  = "odigos.io/data-collection"
	// Label used to identify the Odigos pod which is acting as a cluster collector.
	OdigosClusterCollectorLabel = "odigos.io/gateway"
	OdigosClusterCollectorCollectorGroupName = OdigosClusterCollectorDeploymentName

	OdigosNodeCollectorConfigMapName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName = OdigosNodeCollectorDaemonSetName
)
