package instrumentation

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/profiles/profile"
)

var AllowConcurrentAgents = profile.Profile{
	ProfileName:      common.ProfileName("allow_concurrent_agents"),
	MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
	ShortDescription: "This profile allows Odigos to run concurrently with other agents",
	ModifyConfigFunc: func(c *common.OdigosConfiguration) {
		if c.AllowConcurrentAgents == nil {
			allowConcurrentAgents := true
			c.AllowConcurrentAgents = &allowConcurrentAgents
		}
	},
}
