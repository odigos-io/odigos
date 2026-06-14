package k8sconsts

// Names and image references for the Odigos Security service.
// Defined in OSS even though the image lives in the enterprise registry,
// so the Helm chart in this repo can reference these constants directly.
const (
	OdigosSecurityAppName            = "odigos-security"
	OdigosSecurityDeploymentName     = "odigos-security"
	OdigosSecurityServiceAccountName = "odigos-security"
	OdigosSecurityRoleName           = "odigos-security"
	OdigosSecurityRoleBindingName    = "odigos-security"
	OdigosSecurityServiceName        = "odigos-security"
	OdigosSecurityContainerName      = "odigos-security"

	OdigosSecurityImageName = "odigos-enterprise-security"

	OdigosSecurityHTTPPort = 8080
	OdigosSecurityOTLPPort = 4317
)
