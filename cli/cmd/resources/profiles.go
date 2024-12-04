package resources

import (
	"context"
	"fmt"

	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/cmd/resources/profiles"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
)

type Profile struct {
	ProfileName      common.ProfileName
	ShortDescription string
	KubeObject       kube.Object          // used to read it from the embedded YAML file
	Dependencies     []common.ProfileName // other profiles that are applied by the current profile
}

var (
	// sizing profiles for the collector gateway
	sizeSProfile = Profile{
		ProfileName:      common.ProfileName("size_s"),
		ShortDescription: "Small size deployment profile",
	}
	sizeMProfile = Profile{
		ProfileName:      common.ProfileName("size_m"),
		ShortDescription: "Medium size deployment profile",
	}
	sizeLProfile = Profile{
		ProfileName:      common.ProfileName("size_l"),
		ShortDescription: "Large size deployment profile",
	}
	fullPayloadCollectionProfile = Profile{
		ProfileName:      common.ProfileName("full-payload-collection"),
		ShortDescription: "Collect any payload from the cluster where supported with default settings",
		KubeObject:       &odigosv1alpha1.InstrumentationRule{},
	}
	dbPayloadCollectionProfile = Profile{
		ProfileName:      common.ProfileName("db-payload-collection"),
		ShortDescription: "Collect db payload from the cluster where supported with default settings",
		KubeObject:       &odigosv1alpha1.InstrumentationRule{},
	}
	queryOperationDetector = Profile{
		ProfileName:      common.ProfileName("query-operation-detector"),
		ShortDescription: "Detect the SQL operation name from the query text",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	semconvUpgraderProfile = Profile{
		ProfileName:      common.ProfileName("semconv"),
		ShortDescription: "Upgrade and align some attribute names to a newer version of the OpenTelemetry semantic conventions",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	categoryAttributesProfile = Profile{
		ProfileName:      common.ProfileName("category-attributes"),
		ShortDescription: "Add category attributes to the spans",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	copyScopeProfile = Profile{
		ProfileName:      common.ProfileName("copy-scope"),
		ShortDescription: "Copy the scope name into a separate attribute for backends that do not support scopes",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	hostnameAsPodNameProfile = Profile{
		ProfileName:      common.ProfileName("hostname-as-podname"),
		ShortDescription: "Populate the spans resource `host.name` attribute with value of `k8s.pod.name`",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	javaNativeInstrumentationsProfile = Profile{
		ProfileName:      common.ProfileName("java-native-instrumentations"),
		ShortDescription: "Instrument Java applications using native instrumentation and eBPF enterprise processing",
		KubeObject:       &odigosv1alpha1.InstrumentationRule{},
	}
	codeAttributesProfile = Profile{
		ProfileName:      common.ProfileName("code-attributes"),
		ShortDescription: "Record span attributes in 'code' namespace where supported",
	}
	disableNameProcessorProfile = Profile{
		ProfileName:      common.ProfileName("disable-name-processor"),
		ShortDescription: "If not using dotnet or java native instrumentations, disable the name processor which is not needed",
	}
	smallBatchesProfile = Profile{
		ProfileName:      common.ProfileName("small-batches"),
		ShortDescription: "Reduce the batch size for exports",
		KubeObject:       &odigosv1alpha1.Processor{},
	}
	kratosProfile = Profile{
		ProfileName:      common.ProfileName("kratos"),
		ShortDescription: "Bundle profile that includes db-payload-collection, semconv, category-attributes, copy-scope, hostname-as-podname, java-native-instrumentations, code-attributes, query-operation-detector, disableNameProcessorProfile, small-batches, size_m",
		Dependencies:     []common.ProfileName{"db-payload-collection", "semconv", "category-attributes", "copy-scope", "hostname-as-podname", "java-native-instrumentations", "code-attributes", "query-operation-detector", "disableNameProcessorProfile", "small-batches", "size_m"},
	}
	profilesMap = map[common.ProfileName]Profile{
		sizeSProfile.ProfileName:                      sizeSProfile,
		sizeMProfile.ProfileName:                      sizeMProfile,
		sizeLProfile.ProfileName:                      sizeLProfile,
		fullPayloadCollectionProfile.ProfileName:      fullPayloadCollectionProfile,
		dbPayloadCollectionProfile.ProfileName:        dbPayloadCollectionProfile,
		queryOperationDetector.ProfileName:            queryOperationDetector,
		semconvUpgraderProfile.ProfileName:            semconvUpgraderProfile,
		categoryAttributesProfile.ProfileName:         categoryAttributesProfile,
		copyScopeProfile.ProfileName:                  copyScopeProfile,
		hostnameAsPodNameProfile.ProfileName:          hostnameAsPodNameProfile,
		javaNativeInstrumentationsProfile.ProfileName: javaNativeInstrumentationsProfile,
		codeAttributesProfile.ProfileName:             codeAttributesProfile,
		disableNameProcessorProfile.ProfileName:       disableNameProcessorProfile,
		smallBatchesProfile.ProfileName:               smallBatchesProfile,
		kratosProfile.ProfileName:                     kratosProfile,
	}
)

func GetAvailableCommunityProfiles() []Profile {
	return []Profile{semconvUpgraderProfile, copyScopeProfile, disableNameProcessorProfile, sizeSProfile, sizeMProfile,
		sizeLProfile}
}

func GetAvailableOnPremProfiles() []Profile {
	return append([]Profile{fullPayloadCollectionProfile, dbPayloadCollectionProfile, categoryAttributesProfile,
		hostnameAsPodNameProfile, javaNativeInstrumentationsProfile, kratosProfile, queryOperationDetector,
		smallBatchesProfile},
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

func FilterSizeProfiles(profiles []common.ProfileName) common.ProfileName {
	// In case multiple size profiles are provided, the first one will be used.

	for _, profile := range profiles {
		// Check if the profile is a size profile.
		switch profile {
		case sizeSProfile.ProfileName, sizeMProfile.ProfileName, sizeLProfile.ProfileName:
			return profile
		}

		// Check if the profile has a dependency which is a size profile.
		profileDependencies := profilesMap[profile].Dependencies
		for _, dependencyProfile := range profileDependencies {
			switch dependencyProfile {
			case sizeSProfile.ProfileName, sizeMProfile.ProfileName, sizeLProfile.ProfileName:
				return dependencyProfile
			}
		}
	}
	return ""
}
