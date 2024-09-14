package resources

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/profiles"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Profile struct {
	ProfileName      common.ProfileName
	ShortDescription string
}

var (
	fullPayloadCollectionProfile = Profile{
		ProfileName:      common.ProfileName("full-payload-collection"),
		ShortDescription: "Collect any payload from the cluster where supported with default settings",
	}
	semconvUpgraderProfile = Profile{
		ProfileName:      common.ProfileName("semconv"),
		ShortDescription: "Upgrade and align some attribute names to a newer version of the OpenTelemetry semantic conventions",
	}
	categoryAttributesProfile = Profile{
		ProfileName:      common.ProfileName("category-attributes"),
		ShortDescription: "Add category attributes to the spans",
	}
)

func GetAvailableCommunityProfiles() []Profile {
	return []Profile{semconvUpgraderProfile}
}

func GetAvailableOnPremProfiles() []Profile {
	return append([]Profile{fullPayloadCollectionProfile, categoryAttributesProfile},
		GetAvailableCommunityProfiles()...)
}

func GetResourcesForProfileName(profileName string) ([]client.Object, error) {
	switch profileName {
	case "full-payload-collection":
		return profiles.GetEmbeddedYAMLInstrumentationRuleFileAsObjects("full-payload-collection.yaml")
	case "semconv":
		return profiles.GetEmbeddedYAMLRenameAttributeActionFileAsObjects("semconv.yaml")
	case "category-attributes":
		return profiles.GetEmbeddedYAMLProcessorFileAsObjects("category-attributes.yaml")
	}
	return nil, nil
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

func (a *profilesResourceManager) Name() string { return "Profiles" }

func (a *profilesResourceManager) InstallFromScratch(ctx context.Context) error {
	allResources := []client.Object{}
	for _, profile := range a.config.Profiles {
		profileResources, err := GetResourcesForProfileName(string(profile))
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
