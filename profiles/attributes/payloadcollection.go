package attributes

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var FullPayloadCollectionProfile = profile.Profile{
	ProfileName:      common.ProfileName("full-payload-collection"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Collect any payload from the cluster where supported with default settings",
	KubeObject:       &odigosv1alpha1.InstrumentationRule{},
}

var DbPayloadCollectionProfile = profile.Profile{
	ProfileName:      common.ProfileName("db-payload-collection"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Collect db payload from the cluster where supported with default settings",
	KubeObject:       &odigosv1alpha1.InstrumentationRule{},
}
