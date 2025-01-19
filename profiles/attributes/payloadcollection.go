package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var FullPayloadCollectionProfile = profile.Profile{
	ProfileName:      common.ProfileName("full-payload-collection"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Collect any payload from the cluster where supported with default settings",
}

var DbPayloadCollectionProfile = profile.Profile{
	ProfileName:      common.ProfileName("db-payload-collection"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Collect db payload from the cluster where supported with default settings",
}
