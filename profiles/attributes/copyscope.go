package attributes

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var CopyScopeProfile = profile.Profile{
	ProfileName:      common.ProfileName("copy-scope"),
	MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
	ShortDescription: "Copy the scope name into a separate attribute for backends that do not support scopes",
	KubeObject:       &odigosv1alpha1.Processor{},
}
