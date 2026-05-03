package k8sconsts

import (
	"k8s.io/apimachinery/pkg/util/version"
)

const (
	RedHatImagePrefix = "registry.connect.redhat.com/odigos"
	OdigosImagePrefix = "registry.odigos.io"
)

const (
	DefaultPprofEndpointPort      int32 = 6060
	DevicePluginPprofEndpointPort int32 = 6061
	CollectorsPprofEndpointPort   int32 = 1777
)

const (
	OdigosUiServiceName = "ui"
	OdigosUiServicePort = 3000
)

// MinK8SVersionForInstallation is the minimum Kubernetes version required for Odigos installation
// this value must be in sync with the one defined in the kubeVersion field in Chart.yaml
var MinK8SVersionForInstallation = version.MustParse("v1.20.15-0")

var (
	// Default ignored namespaces
	DefaultIgnoredNamespaces = []string{
		"local-path-storage",
		"istio-system",
		"linkerd",
		"kube-node-lease",
	}

	// Openshift namespaces
	OpenshiftIgnoredNamespaces = []string{
		"openshift",
		"openshift-apiserver",
		"openshift-apiserver-operator",
		"openshift-authentication",
		"openshift-authentication-operator",
		"openshift-cloud-network-config-controller",
		"openshift-cloud-platform-infra",
		"openshift-cluster-machine-approver",
		"openshift-cluster-samples-operator",
		"openshift-cluster-storage-operator",
		"openshift-cluster-version",
		"openshift-config",
		"openshift-config-managed",
		"openshift-config-operator",
		"openshift-console",
		"openshift-console-operator",
		"openshift-console-user-settings",
		"openshift-controller-manager",
		"openshift-controller-manager-operator",
		"openshift-dns",
		"openshift-dns-operator",
		"openshift-etcd",
		"openshift-etcd-operator",
		"openshift-host-network",
		"openshift-image-registry",
		"openshift-infra",
		"openshift-ingress",
		"openshift-ingress-canary",
		"openshift-ingress-operator",
		"openshift-kni-infra",
		"openshift-kube-apiserver",
		"openshift-kube-apiserver-operator",
		"openshift-kube-controller-manager",
		"openshift-kube-controller-manager-operator",
		"openshift-kube-scheduler",
		"openshift-kube-scheduler-operator",
		"openshift-kube-storage-version-migrator",
		"openshift-kube-storage-version-migrator-operator",
		"openshift-machine-api",
		"openshift-machine-config-operator",
		"openshift-marketplace",
		"openshift-monitoring",
		"openshift-multus",
		"openshift-network-console",
		"openshift-network-diagnostics",
		"openshift-network-node-identity",
		"openshift-network-operator",
		"openshift-node",
		"openshift-nutanix-infra",
		"openshift-oauth-apiserver",
		"openshift-openstack-infra",
		"openshift-operator-lifecycle-manager",
		"openshift-operators",
		"openshift-ovirt-infra",
		"openshift-ovn-kubernetes",
		"openshift-route-controller-manager",
		"openshift-service-ca",
		"openshift-service-ca-operator",
		"openshift-user-workload-monitoring",
		"openshift-vsphere-infra",
	}

	// Default ignored container names
	DefaultIgnoredContainers = []string{"istio-proxy", "vault-agent", "filebeat", "linkerd-proxy", "fluentd", "akeyless-init"}
)

const (
	// Custom resource attribute for Argo Rollouts workload name.
	// There is no semconv key for Argo Rollouts, so we use this custom key with argoproj prefix.
	K8SArgoRolloutNameAttribute = "k8s.argoproj.rollout.name"
)
