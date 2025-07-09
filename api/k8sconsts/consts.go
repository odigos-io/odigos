package k8sconsts

import (
	"k8s.io/apimachinery/pkg/util/version"
)

const RedHatImagePrefix = "registry.connect.redhat.com/odigos"
const OdigosImagePrefix = "registry.odigos.io"

const (
	OdigletPprofEndpointPort    int32 = 6060
	CollectorsPprofEndpointPort int32 = 1777
)

const (
	OdigosUiServiceName = "ui"
	OdigosUiServicePort = 3000
)

var (
	// MinK8SVersionForInstallation is the minimum Kubernetes version required for Odigos installation
	// this value must be in sync with the one defined in the kubeVersion field in Chart.yaml
	MinK8SVersionForInstallation = version.MustParse("v1.20.15-0")
)

var (
	DefaultIgnoredNamespaces = []string{"kube-system", "local-path-storage", "istio-system", "linkerd", "kube-node-lease"}
	DefaultIgnoredContainers = []string{"istio-proxy", "vault-agent", "filebeat", "linkerd-proxy", "fluentd", "akeyless-init"}
)
