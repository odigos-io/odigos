package resources

import (
	"github.com/odigos-io/odigos/cli/cmd/resources/centralodigos"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
)

func CreateCentralizedManagers(client *kube.Client, managerOpts resourcemanager.ManagerOpts, ns string, odigosVersion string) []resourcemanager.ResourceManager {
	return []resourcemanager.ResourceManager{
		centralodigos.NewRedisResourceManager(client, ns, managerOpts),
		centralodigos.NewCentralUIResourceManager(client, ns, managerOpts, odigosVersion),
		centralodigos.NewCentralBackendResourceManager(client, ns, odigosVersion, managerOpts),
	}
}
