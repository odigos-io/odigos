package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var SemconvRedisProfile = profile.Profile{
	ProfileName:      common.ProfileName("semconvredis"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Convert db.system.name to db.system for Redis spans",
}
