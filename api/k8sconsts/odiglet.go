package k8sconsts

const (
	OdigletDaemonSetName          = "odiglet"
	OdigletAppLabelValue          = OdigletDaemonSetName
	OdigletServiceAccountName     = OdigletDaemonSetName
	OdigletRoleName               = OdigletDaemonSetName
	OdigletRoleBindingName        = OdigletDaemonSetName
	OdigletClusterRoleName        = OdigletDaemonSetName
	OdigletClusterRoleBindingName = OdigletDaemonSetName
	OdigletContainerName          = "odiglet"
	OdigletImageName              = "keyval/odigos-odiglet"
	OdigletEnterpriseImageName    = "keyval/odigos-enterprise-odiglet"
	OdigletEnterpriseImageUBI9    = "odigos-enterprise-odiglet-ubi9"
	OdigletImageUBI9              = "odigos-odiglet-ubi9"
)
