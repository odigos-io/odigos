package k8sconsts

import (
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	RedHatImagePrefix = "registry.connect.redhat.com/odigos"
	OdigosImagePrefix = "registry.odigos.io"
)

const (
	DefaultDebugPort      int32 = 6060
	DevicePluginDebugPort int32 = 6061
	CollectorsDebugPort   int32 = 1777
)

const (
	OdigosUiServiceName = "ui"
	OdigosUiServicePort = 3000
)

// MinK8SVersionForInstallation is the minimum Kubernetes version required for Odigos installation
// this value must be in sync with the one defined in the kubeVersion field in Chart.yaml
var MinK8SVersionForInstallation = version.MustParse("v1.20.15-0")

var (
	DefaultIgnoredNamespaces = []string{"local-path-storage", "istio-system", "linkerd", "kube-node-lease"}
	DefaultIgnoredContainers = []string{"istio-proxy", "vault-agent", "filebeat", "linkerd-proxy", "fluentd", "akeyless-init"}
)

const (
	// Custom resource attribute for Argo Rollouts workload name.
	// There is no semconv key for Argo Rollouts, so we use this custom key with argoproj prefix.
	K8SArgoRolloutNameAttribute = "k8s.argoproj.rollout.name"
)
