package resources

import (
	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/cli/pkg/kube"
)

// set apiKey to nil for no-op.
// set to empty string for "no api key" (non odigos cloud mode).
// set to a valid api key for odigos cloud mode.
func CreateResourceManagers(client *kube.Client, odigosNs string, isOdigosCloud bool, apiKey *string, config *odigosv1.OdigosConfigurationSpec) []ResourceManager {

	// Note - the order of resource managers is important.
	// If resource A depends on resource B, then A must be installed after B.
	resourceManagers := []ResourceManager{
		NewOdigosDeploymentResourceManager(client, odigosNs, config),
		NewOdigosConfigResourceManager(client, odigosNs, config),
	}

	if isOdigosCloud {
		resourceManagers = append(resourceManagers, NewOdigosCloudResourceManager(client, odigosNs, config, apiKey))
	}

	resourceManagers = append(resourceManagers, []ResourceManager{
		NewOwnTelemetryResourceManager(client, odigosNs, config, isOdigosCloud),
		NewDataCollectionResourceManager(client, odigosNs, config),
		NewInstrumentorResourceManager(client, odigosNs, config),
		NewSchedulerResourceManager(client, odigosNs, config),
		NewOdigletResourceManager(client, odigosNs, config),
		NewAutoScalerResourceManager(client, odigosNs, config),
	}...)

	if isOdigosCloud {
		resourceManagers = append(resourceManagers, NewKeyvalProxyResourceManager(client, odigosNs, config))
	}

	return resourceManagers
}
