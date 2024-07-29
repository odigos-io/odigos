package resources

import (
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
)

// set apiKey to nil for no-op.
// set to empty string for "no api key" (non odigos cloud mode).
// set to a valid api key for odigos cloud mode.
func CreateResourceManagers(client *kube.Client, odigosNs string, odigosTier common.OdigosTier, proTierToken *string, config *common.OdigosConfiguration) []resourcemanager.ResourceManager {

	// Note - the order of resource managers is important.
	// If resource B depends on resource A, then B must be installed after A.
	resourceManagers := []resourcemanager.ResourceManager{
		NewOdigosDeploymentResourceManager(client, odigosNs, config, odigosTier),
		NewOdigosConfigResourceManager(client, odigosNs, config, odigosTier),
	}

	if odigosTier != common.CommunityOdigosTier {
		resourceManagers = append(resourceManagers, odigospro.NewOdigosProResourceManager(client, odigosNs, config, odigosTier, proTierToken))
	}

	// odigos core components are installed for all tiers.
	resourceManagers = append(resourceManagers, []resourcemanager.ResourceManager{
		NewOwnTelemetryResourceManager(client, odigosNs, config, odigosTier),
		NewDataCollectionResourceManager(client, odigosNs, config),
		NewInstrumentorResourceManager(client, odigosNs, config),
		NewSchedulerResourceManager(client, odigosNs, config),
		NewOdigletResourceManager(client, odigosNs, config, odigosTier),
		NewAutoScalerResourceManager(client, odigosNs, config),
		NewUIResourceManager(client, odigosNs, config),
	}...)

	if odigosTier == common.CloudOdigosTier {
		resourceManagers = append(resourceManagers, NewKeyvalProxyResourceManager(client, odigosNs, config))
	}

	return resourceManagers
}
