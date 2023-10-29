package resources

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
)

func CreateResourceManagers(client *kube.Client, odigosNs string, version string, isOdigosCloud bool, config *odigosv1.OdigosConfigurationSpec) []ResourceManager {

	// Note - the order is important.
	// If resource A depends on resource B, then A must be installed after B.
	resourceManager := []ResourceManager{
		NewOwnTelemetryResourceManager(client, odigosNs, version, isOdigosCloud),
		NewOdigosDeploymentResourceManager(client, odigosNs, version),
		NewDataCollectionResourceManager(client, odigosNs, version, config),
		NewInstrumentorResourceManager(client, odigosNs, version, config),
		NewSchedulerResourceManager(client, odigosNs, version),
		NewOdigletResourceManager(client, odigosNs, version, config),
		NewAutoScalerResourceManager(client, odigosNs, version, config),
	}

	if isOdigosCloud {
		resourceManager = append(resourceManager, NewKeyvalProxyResourceManager(client, odigosNs, version))
	}

	return resourceManager
}
