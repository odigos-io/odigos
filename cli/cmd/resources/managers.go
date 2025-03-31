package resources

import (
	"github.com/odigos-io/odigos/cli/cmd/resources/centralodigos"
	"github.com/odigos-io/odigos/cli/cmd/resources/odigospro"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/installationmethod"
)

// set apiKey to nil for no-op.
// set to empty string for "no api key" (non odigos cloud mode).
// set to a valid api key for odigos cloud mode.
func CreateResourceManagers(client *kube.Client, odigosNs string, odigosTier common.OdigosTier, proTierToken *string, config *common.OdigosConfiguration, odigosVersion string, installationMethod installationmethod.K8sInstallationMethod, managerOpts resourcemanager.ManagerOpts) []resourcemanager.ResourceManager {

	// Note - the order of resource managers is important.
	// If resource B depends on resource A, then B must be installed after A.
	resourceManagers := []resourcemanager.ResourceManager{
		NewOdigosDeploymentResourceManager(client, odigosNs, config, odigosTier, odigosVersion, installationMethod, managerOpts),
		NewOdigosConfigResourceManager(client, odigosNs, config, odigosTier, managerOpts),
	}

	if odigosTier != common.CommunityOdigosTier {
		resourceManagers = append(resourceManagers, odigospro.NewOdigosProResourceManager(client, odigosNs, config, odigosTier, proTierToken, managerOpts))
	}

	if managerOpts.IncludeCentralProxy {
		resourceManagers = append(resourceManagers, centralodigos.NewCentralProxyResourceManager(client, odigosNs, config, managerOpts))
	}

	// odigos core components are installed for all tiers.
	resourceManagers = append(resourceManagers, []resourcemanager.ResourceManager{
		NewOwnTelemetryResourceManager(client, odigosNs, config, odigosTier, odigosVersion, managerOpts),
		NewDataCollectionResourceManager(client, odigosNs, config, managerOpts),
		NewInstrumentorResourceManager(client, odigosNs, config, odigosTier, odigosVersion, managerOpts),
		NewSchedulerResourceManager(client, odigosNs, config, odigosVersion, managerOpts),
		NewOdigletResourceManager(client, odigosNs, config, odigosTier, odigosVersion, managerOpts),
		NewAutoScalerResourceManager(client, odigosNs, config, odigosVersion, managerOpts),
		NewUIResourceManager(client, odigosNs, config, odigosVersion, managerOpts),
	}...)

	if odigosTier == common.CloudOdigosTier {
		resourceManagers = append(resourceManagers, NewKeyvalProxyResourceManager(client, odigosNs, config, managerOpts))
	}

	return resourceManagers
}

func CreateCentralizedManagers(client *kube.Client, managerOpts resourcemanager.ManagerOpts) []resourcemanager.ResourceManager {
	ns := consts.DefaultOdigosCentralNamespace

	return []resourcemanager.ResourceManager{
		centralodigos.NewRedisResourceManager(client, ns, managerOpts),
		centralodigos.NewCentralUIResourceManager(client, ns, managerOpts),
		centralodigos.NewCentralBackendResourceManager(client, ns, managerOpts),
	}
}
