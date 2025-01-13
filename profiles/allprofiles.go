package profiles

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/aggregators"
	"github.com/odigos-io/odigos/profiles/attributes"
	"github.com/odigos-io/odigos/profiles/instrumentation"
	"github.com/odigos-io/odigos/profiles/pipeline"
	"github.com/odigos-io/odigos/profiles/profile"
	"github.com/odigos-io/odigos/profiles/sizing"
)

var AllProfiles = []profile.Profile{

	aggregators.KratosProfile,

	attributes.CategoryAttributesProfile,
	attributes.CodeAttributesProfile,
	attributes.CopyScopeProfile,
	attributes.HostnameAsPodNameProfile,
	attributes.FullPayloadCollectionProfile,
	attributes.DbPayloadCollectionProfile,
	attributes.QueryOperationDetector,
	attributes.SemconvUpgraderProfile,

	instrumentation.AllowConcurrentAgents,
	instrumentation.JavaEbpfInstrumentationsProfile,
	instrumentation.JavaNativeInstrumentationsProfile,

	pipeline.DisableNameProcessorProfile,
	pipeline.SmallBatchesProfile,

	sizing.SizeSProfile,
	sizing.SizeMProfile,
	sizing.SizeLProfile,
}

var ProfilesByName = map[common.ProfileName]profile.Profile{}
var CommunityProfiles = []profile.Profile{}
var OnPremProfiles = []profile.Profile{}

func init() {
	for _, p := range AllProfiles {
		ProfilesByName[p.ProfileName] = p
	}
	for _, p := range AllProfiles {
		if p.MinimumTier == common.CommunityOdigosTier {
			// community profiles are also on-prem profiles
			CommunityProfiles = append(CommunityProfiles, p)
			OnPremProfiles = append(OnPremProfiles, p)
		} else if p.MinimumTier == common.OnPremOdigosTier {
			OnPremProfiles = append(OnPremProfiles, p)
		}
	}
}

func GetAvailableProfilesForTier(odigosTier common.OdigosTier) []profile.Profile {
	switch odigosTier {
	case common.CommunityOdigosTier:
		return CommunityProfiles
	case common.OnPremOdigosTier:
		return OnPremProfiles
	default:
		return []profile.Profile{}
	}
}
