package attributes

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var CategoryAttributesProfile = profile.Profile{
	ProfileName:      common.ProfileName("category-attributes"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Add category attributes to the spans",
	KubeObject:       &odigosv1alpha1.Processor{},
}
