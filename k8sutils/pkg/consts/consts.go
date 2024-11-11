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

	// The cluster gateway collector runs as a deployment and the pod is exposed as a service.
	// Thus it cannot collide with other ports on the same node, and we can use an handy default port.
	OdigosClusterCollectorOwnTelemetryPortDefault = int32(8888)
)

const (
	OdigosNodeCollectorDaemonSetName           = "odigos-data-collection"
	OdigosNodeCollectorConfigMapName           = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorCollectorGroupName      = OdigosNodeCollectorDaemonSetName
	OdigosNodeCollectorOwnTelemetryPortDefault = int32(55682)

	OdigosNodeCollectorConfigMapKey = "conf" // this key is different than the cluster collector value. not sure why
)

const (
	OdigosEnvVarNamespace     = "ODIGOS_WORKLOAD_NAMESPACE"
	OdigosEnvVarContainerName = "ODIGOS_CONTAINER_NAME"
	OdigosEnvVarPodName       = "ODIGOS_POD_NAME"
)

func OdigosInjectedEnvVars() []string {
	return []string{
		OdigosEnvVarNamespace,
		OdigosEnvVarContainerName,
		OdigosEnvVarPodName,
	}
}
