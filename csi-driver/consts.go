package main

const (
	// Driver identification
	DriverName    = "odigos.csi.driver"
	DriverVersion = "0.1.0"

	// Socket and endpoint paths
	CSISocketPath            = "/csi/csi.sock"
	CSIEndpoint              = "unix:///csi/csi.sock"
	RegistrationPath         = "/registration"
	RegistrationSocketSuffix = "-reg.sock"

	// Host paths that CSI driver needs access to
	KubeletDir                = "/var/lib/kubelet"
	KubeletPluginsRegistryDir = "/var/lib/kubelet/plugins_registry"
	OdigosAgentsDir           = "/var/odigos"
	ProcMountsPath            = "/proc/mounts"

	// Kubelet CSI plugin paths
	KubeletPluginDir    = "/var/lib/kubelet/plugins/odigos.csi.driver"
	KubeletPluginSocket = "/var/lib/kubelet/plugins/odigos.csi.driver/csi.sock"

	// Environment variables
	NodeNameEnvVar = "NODE_NAME"
)
