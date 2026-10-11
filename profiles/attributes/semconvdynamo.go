package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var SemconvDynamoProfile = profile.Profile{
	ProfileName:      common.ProfileName("semconvdynamo"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Convert db.system.name to db.system for AWS DynamoDB spans",
}
