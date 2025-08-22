package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var TransformDynamoProfile = profile.Profile{
	ProfileName:      common.ProfileName("transformdynamo"),
	MinimumTier:      common.CommunityOdigosTier,
	ShortDescription: "Convert aws.dynamodb.table_names to db.collection.name for AWS DynamoDB spans",
}
