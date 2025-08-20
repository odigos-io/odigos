package sizing

import "github.com/odigos-io/odigos/common"

type SizingConfig struct {
	CollectorGatewayConfig common.CollectorGatewayConfiguration
	CollectorNodeConfig    common.CollectorNodeConfiguration
}

type Sizing string

const (
	SizeSmall  Sizing = "size_s"
	SizeMedium Sizing = "size_m"
	SizeLarge  Sizing = "size_l"
)

var configs = map[Sizing]SizingConfig{
	SizeSmall: {
		CollectorGatewayConfig: common.CollectorGatewayConfiguration{
			MinReplicas:      1,
			MaxReplicas:      5,
			RequestCPUm:      150,
			LimitCPUm:        300,
			RequestMemoryMiB: 300,
			LimitMemoryMiB:   300},
		CollectorNodeConfig: common.CollectorNodeConfiguration{
			RequestMemoryMiB: 150,
			LimitMemoryMiB:   300,
			RequestCPUm:      150,
			LimitCPUm:        300,
		},
	},
	SizeMedium: {
		CollectorGatewayConfig: common.CollectorGatewayConfiguration{
			MinReplicas:      2,
			MaxReplicas:      8,
			RequestCPUm:      500,
			LimitCPUm:        1000,
			RequestMemoryMiB: 500,
			LimitMemoryMiB:   600,
		},
		CollectorNodeConfig: common.CollectorNodeConfiguration{
			RequestMemoryMiB: 250,
			LimitMemoryMiB:   500,
			RequestCPUm:      250,
			LimitCPUm:        500,
		},
	},
	SizeLarge: {
		CollectorGatewayConfig: common.CollectorGatewayConfiguration{
			MinReplicas:      3,
			MaxReplicas:      12,
			RequestCPUm:      750,
			LimitCPUm:        1250,
			RequestMemoryMiB: 750,
			LimitMemoryMiB:   850,
		},
		CollectorNodeConfig: common.CollectorNodeConfiguration{
			RequestMemoryMiB: 500,
			LimitMemoryMiB:   750,
			RequestCPUm:      500,
			LimitCPUm:        750,
		},
	},
}

func ModifySizingConfig(c *common.OdigosConfiguration) {
	if c.SizingConfig == "" {
		// default to size_m if no sizing config is set
		c.SizingConfig = string(SizeMedium)
	}

	sizingConfig := configs[Sizing(c.SizingConfig)]

	// do not modify the configuration if any of the values if they are already set
	if c.CollectorGateway != nil {
		return
	}
	// the following is not very elegant.
	// we only care if the sizing parameters are set, if the port is set, we apply it nevertheless
	if c.CollectorNode != nil &&
		(c.CollectorNode.RequestMemoryMiB != 0 ||
			c.CollectorNode.LimitMemoryMiB != 0 ||
			c.CollectorNode.RequestCPUm != 0 ||
			c.CollectorNode.LimitCPUm != 0) {
		return
	}

	c.CollectorGateway = &sizingConfig.CollectorGatewayConfig
	c.CollectorNode = &sizingConfig.CollectorNodeConfig
}

var validSizings = map[Sizing]struct{}{
	SizeSmall:  {},
	SizeMedium: {},
	SizeLarge:  {},
}

func IsValidSizing(s string) bool {
	_, ok := validSizings[Sizing(s)]
	return ok
}
