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

const (
	// Resource settings for central components
	CentralCPURequest    = "100m"
	CentralCPULimit      = "500m"
	CentralMemoryRequest = "64Mi"
	CentralMemoryLimit   = "256Mi"
)

const (
	// Redis constants
	RedisResourceManagerName = "Redis"
	RedisAppName             = "redis"
	RedisDeploymentName      = "redis"
	RedisServiceName         = "redis"
	RedisContainerName       = "redis"
	RedisContainerImage      = "redis:7.2.4"
	RedisPort                = 6379
	RedisPortName            = "redis"
	RedisCommand             = "redis-server"
	RedisCommandArgPortKey   = "--port"
)
