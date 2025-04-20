package k8sconsts

const (
	// Central Proxy
	CentralProxyAppName            = "central-proxy"
	CentralProxyDeploymentName     = "central-proxy"
	CentralProxyServiceAccountName = "central-proxy"
	CentralProxyRoleName           = "central-proxy"
	CentralProxyRoleBindingName    = "central-proxy"
	CentralProxyLabelAppNameValue  = "central-proxy"
	CentralProxyContainerName      = "central-proxy"

	CentralProxyContainerPort     = 8080
	CentralProxyConfigMapResource = "configmaps"
	CentralProxyImage             = "odigos-enterprise-central-proxy"
)

const (
	// Resource settings for central components
	CentralCPURequest    = "100m"
	CentralCPULimit      = "500m"
	CentralMemoryRequest = "64Mi"
	CentralMemoryLimit   = "256Mi"
)
