package k8sconsts

const (
	// Central Proxy
	CentralProxyAppName                = "central-proxy"
	CentralProxyDeploymentName         = "central-proxy"
	CentralProxyServiceAccountName     = "central-proxy"
	CentralProxyRoleName               = "central-proxy"
	CentralProxyRoleBindingName        = "central-proxy"
	CentralProxyClusterRoleName        = "central-proxy"
	CentralProxyClusterRoleBindingName = "central-proxy"
	CentralProxyLabelAppNameValue      = "central-proxy"
	CentralProxyContainerName          = "central-proxy"
	CentralProxyContainerPort          = 8080
	CentralProxyConfigMapResource      = "configmaps"
	CentralProxyImage                  = "odigos-enterprise-central-proxy"
)

const (
	// Resource settings for central components
	CentralCPURequest    = "100m"
	CentralCPULimit      = "500m"
	CentralMemoryRequest = "64Mi"
	CentralMemoryLimit   = "256Mi"
)

const (
	// Odigos Central Deployment ConfigMap (installation metadata for upgrades/support)
	OdigosCentralDeploymentConfigMapName                  = "odigos-central-deployment"
	OdigosCentralDeploymentConfigMapVersionKey            = "odigosCentralVersion"
	OdigosCentralDeploymentConfigMapInstallationMethodKey = "installationMethod"
	DefaultOdigosCentralNamespace                         = "odigos-central"
)

const (
	// Odigos Central Backend
	CentralBackendAppName            = "central-backend"
	CentralBackendName               = "central-backend"
	CentralBackendImage              = "odigos-enterprise-central-backend"
	CentralBackendServiceAccountName = "central-backend"
	CentralBackendRoleName           = "central-backend"
	CentralBackendRoleBindingName    = "central-backend"

	// Default CPU utilization target (percentage) for HPA when using CPU-based scaling
	CentralBackendDefaultCpuTargetUtilization = 70
)

const (
	//Odigos Central UI
	CentralUI               = "central-ui"
	CentralUIAppName        = "central-ui"
	CentralUIDeploymentName = "central-ui"
	CentralUIContainerName  = "central-ui"
	CentralUIServiceName    = "central-ui"
	CentralUIImage          = "odigos-enterprise-central-ui"
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
	KeycloakResourceManagerName = "Keycloak"
	KeycloakAppName             = "keycloak"
	KeycloakDeploymentName      = "keycloak"
	KeycloakServiceName         = "keycloak"
	KeycloakContainerName       = "keycloak"
	KeycloakImage               = "quay.io/keycloak/keycloak:24.0.3"
	KeycloakPort                = 8080
	KeycloakPortName            = "http"
	KeycloakSecretName          = "keycloak-admin-credentials"
	KeycloakAdminUsernameKey    = "admin-username"
	KeycloakAdminPasswordKey    = "admin-password"
	KeycloakDataPVCName         = "keycloak-data"
	KeycloakDataVolumeName      = "keycloak-data"
)

const (
	OdigosSystemLabelCentralKey = "odigos.io/central-system-object"
)

const (
	CentralBackendPort = "8081"
	CentralUIPort      = "3000"
)
