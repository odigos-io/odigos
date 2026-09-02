package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var InferDbAttributesProfile = profile.Profile{
	ProfileName:      common.ProfileName("infer-db-attributes"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Parse database query text to add attributes like db.operation.name and db.collection.name when they are missing",
}
