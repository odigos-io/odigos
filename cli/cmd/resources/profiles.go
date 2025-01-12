package resources

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles"
	"github.com/odigos-io/odigos/profiles/manifests"
	"github.com/odigos-io/odigos/profiles/profile"
)

func GetResourcesForProfileName(profileName common.ProfileName, tier common.OdigosTier) ([]profile.K8sObject, error) {
	allAvailableProfiles := GetAvailableProfilesForTier(tier)
	for _, p := range allAvailableProfiles {
		if p.ProfileName == common.ProfileName(profileName) {
			if p.KubeObject != nil {
				filename := fmt.Sprintf("%s.yaml", profileName)
				return manifests.GetEmbeddedResourceManifestsAsObjects(filename, p.KubeObject)
			}
			if len(p.Dependencies) > 0 {
				allResources := []profile.K8sObject{}
				for _, dep := range p.Dependencies {
					resources, err := GetResourcesForProfileName(dep, tier)
					if err != nil {
						return nil, err
					}
					allResources = append(allResources, resources...)
				}
				return allResources, nil
			}
			return nil, nil // a profile might not be implemented as a resource necessarily
		}
	}

	return nil, nil
}

func GetAvailableProfilesForTier(odigosTier common.OdigosTier) []profile.Profile {
	switch odigosTier {
	case common.CommunityOdigosTier:
		return profiles.CommunityProfiles
	case common.OnPremOdigosTier:
		return profiles.OnPremProfiles
	default:
		return []profile.Profile{}
	}
}

type profilesResourceManager struct {
	client *kube.Client
	ns     string
	config *common.OdigosConfiguration
	tier   common.OdigosTier
}

func NewProfilesResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, tier common.OdigosTier) resourcemanager.ResourceManager {
	return &profilesResourceManager{client: client, ns: ns, config: config, tier: tier}
}

func (a *profilesResourceManager) Name() string { return "Profiles" }

func (a *profilesResourceManager) InstallFromScratch(ctx context.Context) error {
	allResources := []kube.Object{}
	for _, profile := range a.config.Profiles {
		profileResources, err := GetResourcesForProfileName(profile, a.tier)
		if err != nil {
			return err
		}
		for _, r := range profileResources {
			r.SetNamespace(a.ns)
			allResources = append(allResources, r)
		}
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, allResources)
}
