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
	CentralProxyContainerPort      = 8080
	CentralProxyConfigMapResource  = "configmaps"
	CentralProxyImage              = "odigos-enterprise-central-proxy"
)

const (
	// Resource settings for central components
	CentralCPURequest    = "100m"
	CentralCPULimit      = "500m"
	CentralMemoryRequest = "64Mi"
	CentralMemoryLimit   = "256Mi"
)

const (
	// Odigos Central Backend
	CentralBackendAppName = "central-backend"
	CentralBackendName    = "central-backend"
	CentralBackendImage   = "odigos-enterprise-central-backend"
)

const (
	//Odigos Central UI
	CentralUI               = "central-ui"
	CentralUIAppName        = "central-ui"
	CentralUIDeploymentName = "central-ui"
	CentralUILabelAppValue  = "central-ui"
	CentralUIContainerName  = "central-ui"
	CentralUIImage          = "odigos-enterprise-central-ui"
)

const (
	// Environment variables used by the Central UI
	EnvNextPublicBackendHTTPURL = "NEXT_PUBLIC_BACKEND_URL"
	EnvNextPublicBackendWSURL   = "NEXT_PUBLIC_BACKEND_WS_URL"
)

const (
	RedisResourceManagerName = "Redis"
	RedisAppName             = "redis"
	RedisDeploymentName      = "redis"
	RedisServiceName         = "redis"
	RedisContainerName       = "redis"
	RedisImage               = "redis:7.4.2"
	RedisPort                = 6379
	RedisPortName            = "redis"
	RedisCommand             = "redis-server"
)

const (
	OdigosSystemLabelCentralKey = "odigos.io/central-system-object"
)
