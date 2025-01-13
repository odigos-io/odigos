package consts

import (
	"k8s.io/apimachinery/pkg/util/version"

	commonconsts "github.com/odigos-io/odigos/common/consts"
)

var (
	DefaultIgnoredNamespaces = []string{"kube-system", "local-path-storage", "istio-system", "linkerd", "kube-node-lease"}
	DefaultIgnoredContainers = []string{"istio-proxy", "vault-agent", "filebeat", "linkerd-proxy", "fluentd", "akeyless-init"}
)

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

	// used to label resources created by profiles with the hash that created them.
	// when a new profiles is reconciled, we will apply them with a new hash
	// and use the label to identify the resources that needs to be deleted.
	OdigosProfilesHashLabel = "odigos.io/profiles-hash"

	// for resources auto created by a profile, this annotation will record
	// the name of the profile that created them.
	OdigosProfileAnnotation = "odigos.io/profile"
)

const (
	OdigosDeploymentConfigMapName                  = "odigos-deployment"
	OdigosDeploymentConfigMapVersionKey            = commonconsts.OdigosVersionEnvVarName
	OdigosDeploymentConfigMapTierKey               = commonconsts.OdigosTierEnvVarName
	OdigosDeploymentConfigMapInstallationMethodKey = "installation-method"
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
	OdigosCloudApiKeySecretKey = "odigos-cloud-api-key"
	OdigosOnpremTokenSecretKey = "odigos-onprem-token"
)
