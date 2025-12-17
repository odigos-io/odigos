package resources

import (
	"github.com/odigos-io/odigos/cli/cmd/resources/centralodigos"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
)

type CentralManagersConfig struct {
	Auth           centralodigos.AuthConfig
	CentralBackend centralodigos.CentralBackendConfig
}

func CreateCentralizedManagers(client *kube.Client, managerOpts resourcemanager.ManagerOpts, ns string, odigosVersion string, config CentralManagersConfig) []resourcemanager.ResourceManager {
	return []resourcemanager.ResourceManager{
		centralodigos.NewRedisResourceManager(client, ns, managerOpts),
		centralodigos.NewKeycloakResourceManager(client, ns, managerOpts, config.Auth),
		centralodigos.NewCentralUIResourceManager(client, ns, managerOpts, odigosVersion),
		centralodigos.NewCentralBackendResourceManager(client, ns, odigosVersion, managerOpts, config.CentralBackend),
	}
}
