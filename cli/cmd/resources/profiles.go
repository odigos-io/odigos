package resources

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/profiles"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
)

type Profile struct {
	ProfileName      common.ProfileName
	ShortDescription string
}

func GetAvailableCommunityProfiles() []Profile {
	return []Profile{}
}

func GetAvailableOnPremProfiles() []Profile {
	return []Profile{
		{
			ProfileName:      common.ProfileName("kratos"),
			ShortDescription: "Includes category attributes",
		},
	}
}

func GetAvailableProfilesForTier(odigosTier common.OdigosTier) []Profile {
	switch odigosTier {
	case common.CommunityOdigosTier:
		return GetAvailableCommunityProfiles()
	case common.OnPremOdigosTier:
		return GetAvailableOnPremProfiles()
	default:
		return []Profile{}
	}
}

type profilesResourceManager struct {
	client *kube.Client
	ns     string
	config *common.OdigosConfiguration
}

func NewProfilesResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration) resourcemanager.ResourceManager {
	return &profilesResourceManager{client: client, ns: ns, config: config}
}

func (a *profilesResourceManager) Name() string { return "CloudProxy" }

func (a *profilesResourceManager) InstallFromScratch(ctx context.Context) error {
	resources, err := profiles.GetEmbeddedYAMLFilesAsObjects()
	if err != nil {
		return err
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}
