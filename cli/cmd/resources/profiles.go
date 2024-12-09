package resources

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/cli/cmd/resources/profiles"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	k8sprofiles "github.com/odigos-io/odigos/k8sutils/pkg/profiles"
)

func GetAvailableCommunityProfiles() []k8sprofiles.Profile {
	return []k8sprofiles.Profile{k8sprofiles.SemconvUpgraderProfile, k8sprofiles.CopyScopeProfile, k8sprofiles.DisableNameProcessorProfile,
		k8sprofiles.SizeSProfile, k8sprofiles.SizeMProfile,
		k8sprofiles.SizeLProfile, k8sprofiles.AllowConcurrentAgents}
}

func GetAvailableOnPremProfiles() []k8sprofiles.Profile {
	return append([]k8sprofiles.Profile{k8sprofiles.FullPayloadCollectionProfile, k8sprofiles.DbPayloadCollectionProfile, k8sprofiles.CategoryAttributesProfile,
		k8sprofiles.HostnameAsPodNameProfile, k8sprofiles.JavaNativeInstrumentationsProfile, k8sprofiles.KratosProfile, k8sprofiles.QueryOperationDetector,
		k8sprofiles.SmallBatchesProfile},
		GetAvailableCommunityProfiles()...)
}

func GetResourcesForProfileName(profileName common.ProfileName, tier common.OdigosTier) ([]kube.Object, error) {
	allAvailableProfiles := GetAvailableProfilesForTier(tier)
	for _, p := range allAvailableProfiles {
		if p.ProfileName == common.ProfileName(profileName) {
			if p.KubeObject != nil {
				filename := fmt.Sprintf("%s.yaml", profileName)
				return profiles.GetEmbeddedYAMLFileAsObjects(filename, p.KubeObject)
			}
			if len(p.Dependencies) > 0 {
				allResources := []kube.Object{}
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

func GetAvailableProfilesForTier(odigosTier common.OdigosTier) []k8sprofiles.Profile {
	switch odigosTier {
	case common.CommunityOdigosTier:
		return GetAvailableCommunityProfiles()
	case common.OnPremOdigosTier:
		return GetAvailableOnPremProfiles()
	default:
		return []k8sprofiles.Profile{}
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
		}
		allResources = append(allResources, profileResources...)
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, allResources)
}
