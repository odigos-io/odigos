package k8sconsts

const (
	CentralUI = "central-ui"
)

const (
	// CentralBackend
	CentralBackendAppName = "central-backend"
	CentralBackendName    = "central-backend"

	// Redis
	CentralBackendRedisEnvName = "REDIS_ADDR"
	CentralBackendRedisAddr    = "redis.odigos-system.svc.cluster.local:6379"
)

const (
	// Central Proxy
	CentralProxyAppName            = "central-proxy"
	CentralProxyDeploymentName     = "central-proxy"
	CentralProxyServiceAccountName = "central-proxy"
	CentralProxyRoleName           = "central-proxy"
	CentralProxyRoleBindingName    = "central-proxy"
	CentralProxyLabelAppNameKey    = "app.kubernetes.io/name"
	CentralProxyLabelAppNameValue  = "central-proxy"
	CentralProxyContainerName      = "central-proxy"
	CentralProxyContainerImage     = "staging-registry.odigos.io/central-proxy:dev"
	CentralProxyContainerPort      = 8080
	CentralProxyRBACAPIGroup       = "rbac.authorization.k8s.io"
	CentralProxyConfigMapResource  = "configmaps"
)

const (
	// Central UI
	CentralUIAppName        = "central-ui"
	CentralUIDeploymentName = "central-ui"
	CentralUILabelAppKey    = "app"
	CentralUILabelAppValue  = "central-ui"
	CentralUIContainerName  = "central-ui"
	CentralUIContainerImage = "staging-registry.odigos.io/central-ui:dev"
)
