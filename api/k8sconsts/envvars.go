package k8sconsts

const (
	OdigosEnvVarNamespace              = "ODIGOS_WORKLOAD_NAMESPACE"
	OdigosEnvVarContainerName          = "ODIGOS_CONTAINER_NAME"
	OdigosEnvVarPodName                = "ODIGOS_POD_NAME"
	OdigosEnvVarDistroName             = "ODIGOS_DISTRO_NAME"
	CustomContainerRuntimeSocketEnvVar = "CONTAINER_RUNTIME_SOCK"
	OtelResourceAttributesEnvVar       = "OTEL_RESOURCE_ATTRIBUTES"
)

func OdigosInjectedEnvVars() []string {
	return []string{
		OdigosEnvVarNamespace,
		OdigosEnvVarContainerName,
		OdigosEnvVarPodName,
		OdigosEnvVarDistroName,
	}
}
