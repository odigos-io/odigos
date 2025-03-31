package k8sconsts

import (
	"fmt"

	"github.com/odigos-io/odigos/common/consts"
)

const (
	CentralUI = "central-ui"
)

const (
	// CentralBackend
	CentralBackendAppName = "central-backend"
	CentralBackendName    = "central-backend"

	// Redis
	CentralBackendRedisEnvName = "REDIS_ADDR"
)

var (
	CentralBackendRedisAddr = fmt.Sprintf("redis.%s.svc.cluster.local:6379", consts.DefaultOdigosCentralNamespace)
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
	CentralProxyContainerImage     = "staging-registry.odigos.io/central-proxy:dev" //TODO: change to odigos registry
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
	CentralUIContainerImage = "staging-registry.odigos.io/central-ui:dev" //TODO: change to odigos registry
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
