package installationmethod

type K8sInstallationMethod string

const (
	K8sInstallationMethodOdigosCli K8sInstallationMethod = "odigos-cli"
	K8sInstallationMethodHelm      K8sInstallationMethod = "helm"
)
