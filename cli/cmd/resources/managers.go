package resources

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
)

func CreateResourceManagers(client *kube.Client, odigosNs string, isOdigosCloud bool, config *odigosv1.OdigosConfigurationSpec) []ResourceManager {

	// Note - the order is important.
	// If resource A depends on resource B, then A must be installed after B.
	resourceManager := []ResourceManager{
		NewOdigosDeploymentResourceManager(client, odigosNs, config),
		NewOdigosConfigResourceManager(client, odigosNs, config),
		NewOwnTelemetryResourceManager(client, odigosNs, config, isOdigosCloud),
		NewDataCollectionResourceManager(client, odigosNs, config),
		NewInstrumentorResourceManager(client, odigosNs, config),
		NewSchedulerResourceManager(client, odigosNs, config),
		NewOdigletResourceManager(client, odigosNs, config),
		NewAutoScalerResourceManager(client, odigosNs, config),
	}

	if isOdigosCloud {
		resourceManager = append(resourceManager, NewKeyvalProxyResourceManager(client, odigosNs, config))
	}

	return resourceManager
}
