package consts

import "k8s.io/apimachinery/pkg/util/version"

type CollectorRole string

const (
	CollectorsRoleClusterGateway CollectorRole = "CLUSTER_GATEWAY"
	CollectorsRoleNodeCollector  CollectorRole = "NODE_COLLECTOR"
)

const (
	// OdigosInjectInstrumentationLabel is the label used to enable the mutating webhook.
	OdigosInjectInstrumentationLabel = "odigos.io/inject-instrumentation"
	// OdigosCollectorRoleLabel is the label used to identify the role of the Odigos collector.
	OdigosCollectorRoleLabel = "odigos.io/collector-role"
)

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
	OdigosProSecretName = "odigos-pro"
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

var (
	// MinK8SVersionForInstallation is the minimum Kubernetes version required for Odigos installation
	// this value must be in sync with the one defined in the kubeVersion field in Chart.yaml
	MinK8SVersionForInstallation = version.MustParse("v1.20.15-0")
)

const (
	// StartLangDetectionFinalizer is used for Workload exclusion Sources. When a Workload exclusion Source
	// is deleted, we want to go to the startlangdetection controller. There, we will check if the Workload should
	// start inheriting Namespace instrumentation.
	StartLangDetectionFinalizer          = "odigos.io/source-startlangdetection-finalizer"
	// DeleteInstrumentationConfigFinalizer is used for all non-exclusion (normal) Sources. When a normal Source
	// is deleted, we want to go to the deleteinstrumentationconfig controller to un-instrument the workload/namespace.
	DeleteInstrumentationConfigFinalizer = "odigos.io/source-deleteinstrumentationconfig-finalizer"

	WorkloadNameLabel      = "odigos.io/workload-name"
	WorkloadNamespaceLabel = "odigos.io/workload-namespace"
	WorkloadKindLabel      = "odigos.io/workload-kind"

	OdigosCloudApiKeySecretKey = "odigos-cloud-api-key"
	OdigosOnpremTokenSecretKey = "odigos-onprem-token"
)
