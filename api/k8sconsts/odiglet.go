package k8sconsts

const (
	OdigletDaemonSetName             = "odiglet"
	OdigletAppLabelValue             = OdigletDaemonSetName
	OdigletServiceAccountName        = OdigletDaemonSetName
	OdigletRoleName                  = OdigletDaemonSetName
	OdigletRoleBindingName           = OdigletDaemonSetName
	OdigletClusterRoleName           = OdigletDaemonSetName
	OdigletClusterRoleBindingName    = OdigletDaemonSetName
	OdigletContainerName             = "odiglet"
	OdigletDevicePluginContainerName = "deviceplugin"
	OdigletImageName                 = "odigos-odiglet"
	OdigletEnterpriseImageName       = "odigos-enterprise-odiglet"
	OdigletEnterpriseImageCertified  = "odigos-enterprise-odiglet-certified"
	OdigletImageCertified            = "odigos-odiglet-certified"

	GrpcHealthProbePath    = "unix:///var/lib/kubelet/device-plugins/instrumentation.odigos.io_generic"
	GrpcHealthBinaryPath   = "/root/grpc_health_probe"
	GrpcHealthProbeTimeout = 10

	// Used to indicate that the odiglet is installed on a node.
	OdigletOSSInstalledLabel          = "odigos.io/odiglet-oss-installed"
	OdigletEnterpriseInstalledLabel   = "odigos.io/odiglet-enterprise-installed"
	OdigletInstalledLabelValue        = "true"
	OdigletDefaultHealthProbeBindPort = 55683

	// ConfigMap used to store custom/updated Go instrumentation offsets
	GoOffsetsConfigMap   = "odigos-go-offsets"
	GoOffsetsFileName    = "go_offset_results.json"
	GoOffsetsEnvVar      = "OTEL_GO_OFFSETS_FILE"
	OffsetFileMountPath  = "/offsets"
	OffsetCronJobName    = "odigos-go-offsets-updater"
	OffsetInitialJobName = "odigos-go-offsets-updater-initial"

	OdigletLocalTrafficServiceName = "odiglet-local"
	OdigletMetricsServerPort       = 8080
	OdigletWaspServicePort         = 4040
)

// OffsetCronJobMode represents the mode for the Go offsets cron job
type OffsetCronJobMode string

const (
	OffsetCronJobModeDirect OffsetCronJobMode = "direct"
	OffsetCronJobModeImage  OffsetCronJobMode = "image"
	OffsetCronJobModeOff    OffsetCronJobMode = "off"
)

// IsValid returns true if the mode is a valid OffsetCronJobMode
func (m OffsetCronJobMode) IsValid() bool {
	switch m {
	case OffsetCronJobModeDirect, OffsetCronJobModeImage, OffsetCronJobModeOff:
		return true
	default:
		return false
	}
}

// String returns the string representation of the mode
func (m OffsetCronJobMode) String() string {
	return string(m)
}

var OdigletOSSInstalled = map[string]string{
	OdigletOSSInstalledLabel: OdigletInstalledLabelValue,
}

var OdigletEnterpriseInstalled = map[string]string{
	OdigletEnterpriseInstalledLabel: OdigletInstalledLabelValue,
}
