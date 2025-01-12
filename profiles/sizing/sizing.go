package sizing

import (
	"github.com/odigos-io/odigos/common"
	profiles "github.com/odigos-io/odigos/profiles/profile"
)

var (
	SizeSProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_s"),
		MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
		ShortDescription: "Small size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas:      1,
					MaxReplicas:      5,
					RequestCPUm:      150,
					LimitCPUm:        300,
					RequestMemoryMiB: 300,
					LimitMemoryMiB:   300,
				},
				common.CollectorNodeConfiguration{
					RequestMemoryMiB: 150,
					LimitMemoryMiB:   300,
					RequestCPUm:      150,
					LimitCPUm:        300,
				})
		},
	}
	SizeMProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_m"),
		MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
		ShortDescription: "Medium size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas:      2,
					MaxReplicas:      8,
					RequestCPUm:      500,
					LimitCPUm:        1000,
					RequestMemoryMiB: 500,
					LimitMemoryMiB:   600,
				},
				common.CollectorNodeConfiguration{
					RequestMemoryMiB: 250,
					LimitMemoryMiB:   500,
					RequestCPUm:      250,
					LimitCPUm:        500,
				})
		},
	}
	SizeLProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_l"),
		MinimumTier:      common.OdigosTier(common.CommunityOdigosTier),
		ShortDescription: "Large size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas:      3,
					MaxReplicas:      12,
					RequestCPUm:      750,
					LimitCPUm:        1250,
					RequestMemoryMiB: 750,
					LimitMemoryMiB:   850,
				},
				common.CollectorNodeConfiguration{
					RequestMemoryMiB: 500,
					LimitMemoryMiB:   750,
					RequestCPUm:      500,
					LimitCPUm:        750,
				})
		},
	}
)

func modifySizingConfig(c *common.OdigosConfiguration, clusterCollectorConfig common.CollectorGatewayConfiguration, nodeCollectorConfig common.CollectorNodeConfiguration) {
	// do not modify the configuration if any of the values if they are already set
	if c.CollectorGateway != nil {
		return
	}
	// the following is not very elegant.
	// we only care if the sizing parameters are set, if the port is set, we apply it nevertheless
	if c.CollectorNode != nil && (c.CollectorNode.RequestMemoryMiB != 0 || c.CollectorNode.LimitMemoryMiB != 0 || c.CollectorNode.RequestCPUm != 0 || c.CollectorNode.LimitCPUm != 0) {
		return
	}

	c.CollectorGateway = &clusterCollectorConfig
	collectorNodeConfig := nodeCollectorConfig
	if c.CollectorNode != nil {
		// make sure we keep the port which is unrelated to the sizing
		collectorNodeConfig.CollectorOwnMetricsPort = c.CollectorNode.CollectorOwnMetricsPort
	}
	c.CollectorNode = &collectorNodeConfig
}
