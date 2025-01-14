package pipeline

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var SmallBatchesProfile = profile.Profile{
	ProfileName:      common.ProfileName("small-batches"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Reduce the batch size for exports",
}
