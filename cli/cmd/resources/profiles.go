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
	copyScopeProfile = Profile{
		ProfileName:      common.ProfileName("copy-scope"),
		ShortDescription: "Copy the scope name into a separate attribute for backends that do not support scopes",
	}
	hostnameAsPodNameProfile = Profile{
		ProfileName:      common.ProfileName("hostname-as-podname"),
		ShortDescription: "Populate the spans resource `host.name` attribute with value of `k8s.pod.name`",
	}

	kratosProfile = Profile{
		ProfileName:      common.ProfileName("kratos"),
		ShortDescription: "Bundle profile that includes full-payload-collection, semconv, category-attributes, copy-scope, hostname-as-podname",
	}
)

func GetAvailableCommunityProfiles() []Profile {
	return []Profile{semconvUpgraderProfile, copyScopeProfile}
}

func GetAvailableOnPremProfiles() []Profile {
	return append([]Profile{fullPayloadCollectionProfile, categoryAttributesProfile, hostnameAsPodNameProfile, kratosProfile},
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
	case "copy-scope":
		return profiles.GetEmbeddedYAMLProcessorFileAsObjects("copy-scope.yaml")
	case "hostname-as-podname":
		return profiles.GetEmbeddedYAMLProcessorFileAsObjects("hostname-as-podname.yaml")
	case "kratos":
		// call and merge all the above profiles
		profiles := []string{"full-payload-collection", "semconv", "category-attributes", "copy-scope", "hostname-as-podname"}
		allResources := []client.Object{}
		for _, p := range profiles {
			resources, err := GetResourcesForProfileName(p)
			if err != nil {
				return nil, err
			}
			allResources = append(allResources, resources...)
		}
		return allResources, nil
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
