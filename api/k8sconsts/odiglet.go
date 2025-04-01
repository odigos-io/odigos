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
	OdigletImageName              = "odigos-odiglet"
	OdigletEnterpriseImageName    = "odigos-enterprise-odiglet"
	OdigletEnterpriseImageUBI9    = "odigos-enterprise-odiglet-ubi9"
	OdigletImageUBI9              = "odigos-odiglet-ubi9"
	// Used to indicate that the odiglet is installed on a node.
	OdigletOSSInstalledLabel        = "odigos.io/odiglet-oss-installed"
	OdigletEnterpriseInstalledLabel = "odigos.io/odiglet-enterprise-installed"
	OdigletInstalledLabelValue      = "true"

	// ConfigMap used to store custom/updated Go instrumentation offsets
	GoOffsetsPublicURL = "https://storage.googleapis.com/odigos-cloud/offset_results_min.json"
	GoOffsetsConfigMap = "odigos-go-offsets"
	GoOffsetsFileName  = "offset_results.json"
	GoOffsetsEnvVar    = "OTEL_GO_OFFSETS_FILE"
)

var OdigletOSSInstalled = map[string]string{
	OdigletOSSInstalledLabel: OdigletInstalledLabelValue,
}

var OdigletEnterpriseInstalled = map[string]string{
	OdigletEnterpriseInstalledLabel: OdigletInstalledLabelValue,
}
