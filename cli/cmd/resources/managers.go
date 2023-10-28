package resources

import "github.com/keyval-dev/odigos/cli/pkg/kube"

func CreateResourceManagers(client *kube.Client, odigosNs string, version string, isOdigosCloud bool, telemetryEnabled bool, sidecarInstrumentation bool, ignoredNamespaces []string, psp bool) []ResourceManager {

	// Note - the order is important.
	// If resource A depends on resource B, then A must be installed after B.
	resourceManager := []ResourceManager{
		NewOwnTelemetryResourceManager(client, odigosNs, version, isOdigosCloud),
		NewOdigosDeploymentResourceManager(client, odigosNs, version),
		NewDataCollectionResourceManager(client, odigosNs, version, psp),
		NewInstrumentorResourceManager(client, odigosNs, version, telemetryEnabled, sidecarInstrumentation, ignoredNamespaces),
		NewSchedulerResourceManager(client, odigosNs, version),
		NewOdigletResourceManager(client, odigosNs, version, psp),
		NewAutoScalerResourceManager(client, odigosNs, version),
	}

	if isOdigosCloud {
		resourceManager = append(resourceManager, NewKeyvalProxyResourceManager(client, odigosNs))
	}

	return resourceManager
}
