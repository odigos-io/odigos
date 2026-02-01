package k8sconsts

// Diagnose command
const (
	LogsDir    = "Logs"
	CRDsDir    = "CRDs"
	ProfileDir = "Profile"
	MetricsDir = "Metrics"
)

const (
	CliImageName        = "odigos-cli"
	CliOffsetsImageName = "odigos-cli-offsets"
)

// Helm constants
const (
	DefaultHelmChart          = "odigos/odigos"
	OdigosHelmRepoName        = "odigos"
	OdigosHelmRepoURL         = "https://odigos-io.github.io/odigos/"
	DefaultCentralHelmChart   = "odigos/odigos-central"
	OdigosCentralHelmRepoName = "odigos-central"
	OdigosCentralHelmRepoURL  = "https://odigos-io.github.io/odigos-central/"
	DefaultCentralReleaseName = "odigos-central"
)
