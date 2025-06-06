package sizing

import (
	"github.com/odigos-io/odigos/common"
	profiles "github.com/odigos-io/odigos/profiles/profile"
)

// Core component resource configurations for each size profile
var (
	// XSmall (75% of small)
	sizeXSCoreResources = common.ResourceConfig{
		RequestMemoryMiB: 48,
		LimitMemoryMiB:   384,
		RequestCPUm:      8,
		LimitCPUm:        375,
	}

	// Small (base size)
	sizeSCoreResources = common.ResourceConfig{
		RequestMemoryMiB: 64,
		LimitMemoryMiB:   512,
		RequestCPUm:      10,
		LimitCPUm:        500,
	}

	// Medium (1.25x small)
	sizeMCoreResources = common.ResourceConfig{
		RequestMemoryMiB: 80,
		LimitMemoryMiB:   640,
		RequestCPUm:      13,
		LimitCPUm:        625,
	}

	// Large (1.5x small)
	sizeLCoreResources = common.ResourceConfig{
		RequestMemoryMiB: 96,
		LimitMemoryMiB:   768,
		RequestCPUm:      15,
		LimitCPUm:        750,
	}
)

var (
	SizeXSProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_xs"),
		MinimumTier:      common.CommunityOdigosTier,
		ShortDescription: "Extra small size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas: 1,
					MaxReplicas: 4,
					ResourceConfig: common.ResourceConfig{
						RequestCPUm:      113,
						LimitCPUm:        225,
						RequestMemoryMiB: 225,
						LimitMemoryMiB:   225,
					},
				},
				common.CollectorNodeConfiguration{
					ResourceConfig: common.ResourceConfig{
						RequestMemoryMiB: 113,
						LimitMemoryMiB:   225,
						RequestCPUm:      113,
						LimitCPUm:        225,
					},
				},
				common.InstrumentorConfiguration{
					ResourceConfig: sizeXSCoreResources,
				},
				common.AutoscalerConfiguration{
					ResourceConfig: sizeXSCoreResources,
				},
				common.SchedulerConfiguration{
					ResourceConfig: sizeXSCoreResources,
				},
				common.UiConfiguration{
					ResourceConfig: sizeXSCoreResources,
				})
		},
	}
	SizeSProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_s"),
		MinimumTier:      common.CommunityOdigosTier,
		ShortDescription: "Small size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas: 1,
					MaxReplicas: 5,
					ResourceConfig: common.ResourceConfig{
						RequestCPUm:      150,
						LimitCPUm:        300,
						RequestMemoryMiB: 300,
						LimitMemoryMiB:   300,
					},
				},
				common.CollectorNodeConfiguration{
					ResourceConfig: common.ResourceConfig{
						RequestMemoryMiB: 150,
						LimitMemoryMiB:   300,
						RequestCPUm:      150,
						LimitCPUm:        300,
					},
				},
				common.InstrumentorConfiguration{
					ResourceConfig: sizeSCoreResources,
				},
				common.AutoscalerConfiguration{
					ResourceConfig: sizeSCoreResources,
				},
				common.SchedulerConfiguration{
					ResourceConfig: sizeSCoreResources,
				},
				common.UiConfiguration{
					ResourceConfig: sizeSCoreResources,
				})
		},
	}
	SizeMProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_m"),
		MinimumTier:      common.CommunityOdigosTier,
		ShortDescription: "Medium size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas: 2,
					MaxReplicas: 8,
					ResourceConfig: common.ResourceConfig{
						RequestCPUm:      500,
						LimitCPUm:        1000,
						RequestMemoryMiB: 500,
						LimitMemoryMiB:   600,
					},
				},
				common.CollectorNodeConfiguration{
					ResourceConfig: common.ResourceConfig{
						RequestMemoryMiB: 250,
						LimitMemoryMiB:   500,
						RequestCPUm:      250,
						LimitCPUm:        500,
					},
				},
				common.InstrumentorConfiguration{
					ResourceConfig: sizeMCoreResources,
				},
				common.AutoscalerConfiguration{
					ResourceConfig: sizeMCoreResources,
				},
				common.SchedulerConfiguration{
					ResourceConfig: sizeMCoreResources,
				},
				common.UiConfiguration{
					ResourceConfig: sizeMCoreResources,
				})
		},
	}
	SizeLProfile = profiles.Profile{
		ProfileName:      common.ProfileName("size_l"),
		MinimumTier:      common.CommunityOdigosTier,
		ShortDescription: "Large size deployment profile",
		ModifyConfigFunc: func(c *common.OdigosConfiguration) {
			modifySizingConfig(c,
				common.CollectorGatewayConfiguration{
					MinReplicas: 3,
					MaxReplicas: 12,
					ResourceConfig: common.ResourceConfig{
						RequestCPUm:      750,
						LimitCPUm:        1250,
						RequestMemoryMiB: 750,
						LimitMemoryMiB:   850,
					},
				},
				common.CollectorNodeConfiguration{
					ResourceConfig: common.ResourceConfig{
						RequestMemoryMiB: 500,
						LimitMemoryMiB:   750,
						RequestCPUm:      500,
						LimitCPUm:        750,
					},
				},
				common.InstrumentorConfiguration{
					ResourceConfig: sizeLCoreResources,
				},
				common.AutoscalerConfiguration{
					ResourceConfig: sizeLCoreResources,
				},
				common.SchedulerConfiguration{
					ResourceConfig: sizeLCoreResources,
				},
				common.UiConfiguration{
					ResourceConfig: sizeLCoreResources,
				})
		},
	}
)

func modifySizingConfig(c *common.OdigosConfiguration,
	clusterCollectorConfig common.CollectorGatewayConfiguration,
	nodeCollectorConfig common.CollectorNodeConfiguration,
	instrumentorConfig common.InstrumentorConfiguration,
	autoscalerConfig common.AutoscalerConfiguration,
	schedulerConfig common.SchedulerConfiguration,
	uiConfig common.UiConfiguration) {
	// Check and apply gateway config if needed
	if c.CollectorGateway == nil || !hasResourceSettings(&c.CollectorGateway.ResourceConfig) {
		c.CollectorGateway = &clusterCollectorConfig
	}

	// Check and apply node collector config if needed
	if c.CollectorNode == nil || !hasResourceSettings(&c.CollectorNode.ResourceConfig) {
		collectorNodeConfig := nodeCollectorConfig
		if c.CollectorNode != nil {
			// make sure we keep values unrelated to sizing
			collectorNodeConfig.CollectorOwnMetricsPort = c.CollectorNode.CollectorOwnMetricsPort
			collectorNodeConfig.K8sNodeLogsDirectory = c.CollectorNode.K8sNodeLogsDirectory
		}
		c.CollectorNode = &collectorNodeConfig
	}

	// Check and apply Instrumentor config if needed
	if c.Instrumentor == nil || !hasResourceSettings(&c.Instrumentor.ResourceConfig) {
		c.Instrumentor = &instrumentorConfig
	}

	// Check and apply Autoscaler config if needed
	if c.Autoscaler == nil || !hasResourceSettings(&c.Autoscaler.ResourceConfig) {
		c.Autoscaler = &autoscalerConfig
	}

	// Check and apply Scheduler config if needed
	if c.Scheduler == nil || !hasResourceSettings(&c.Scheduler.ResourceConfig) {
		c.Scheduler = &schedulerConfig
	}

	// Check and apply UI config if needed
	if c.Ui == nil || !hasResourceSettings(&c.Ui.ResourceConfig) {
		c.Ui = &uiConfig
	}
}

// hasResourceSettings checks if any resource setting is configured in the ResourceConfig
func hasResourceSettings(rc *common.ResourceConfig) bool {
	return rc.RequestMemoryMiB != 0 ||
		rc.LimitMemoryMiB != 0 ||
		rc.RequestCPUm != 0 ||
		rc.LimitCPUm != 0
}
