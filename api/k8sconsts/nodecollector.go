package k8sconsts

const (
	OdigosNodeCollectorDaemonSetName           = "odigos-data-collection"
	OdigosNodeCollectorConfigMapName           = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorOwnTelemetryPortDefault = int32(55682)

	OdigosNodeCollectorLocalTrafficServiceName = "odigos-data-collection-local-traffic"

	OdigosNodeCollectorConfigMapKey = "conf" // this key is different than the cluster collector value. not sure why

	OdigosNodeCollectorServiceAccountName     = "odigos-data-collection"
	OdigosNodeCollectorRoleName               = "odigos-data-collection"
	OdigosNodeCollectorRoleBindingName        = "odigos-data-collection"
	OdigosNodeCollectorClusterRoleName        = "odigos-data-collection"
	OdigosNodeCollectorClusterRoleBindingName = "odigos-data-collection"
)
