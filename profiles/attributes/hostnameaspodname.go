package attributes

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var HostnameAsPodNameProfile = profile.Profile{
	ProfileName:      common.ProfileName("hostname-as-podname"),
	MinimumTier:      common.OdigosTier(common.OnPremOdigosTier),
	ShortDescription: "Populate the spans resource `host.name` attribute with value of `k8s.pod.name`",
}
