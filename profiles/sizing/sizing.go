package sizing

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/odigos-io/odigos/common"
	profiles "github.com/odigos-io/odigos/profiles/profile"
)

var SizeProfilePriority = map[common.ProfileName]int{
	SizeLProfile.ProfileName:  -2,
	SizeMProfile.ProfileName:  -1,
	SizeSProfile.ProfileName:  0,
	SizeXSProfile.ProfileName: 1,
}

var ResourceRequirementsByProfile = map[common.ProfileName]corev1.ResourceRequirements{
	SizeLProfile.ProfileName:  sizeLargeResources,
	SizeMProfile.ProfileName:  sizeMediumResources,
	SizeSProfile.ProfileName:  sizeSmallResources,
	SizeXSProfile.ProfileName: sizeXSmallResources,
}

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
				})
		},
	}
)

// Component resource requirements for each size profile
var (
	// XSmall (75% of small)
	sizeXSmallResources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("375m"),
			"memory": *resource.NewQuantity(402653184, resource.BinarySI), // 384Mi
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("8m"),
			"memory": *resource.NewQuantity(50331648, resource.BinarySI), // 48Mi
		},
	}

	// Small (base size)
	sizeSmallResources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("500m"),
			"memory": *resource.NewQuantity(536870912, resource.BinarySI), // 512Mi
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("10m"),
			"memory": *resource.NewQuantity(67108864, resource.BinarySI), // 64Mi
		},
	}

	// Medium (1.25x small)
	sizeMediumResources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("625m"),
			"memory": *resource.NewQuantity(671088640, resource.BinarySI), // 640Mi
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("13m"),
			"memory": *resource.NewQuantity(83886080, resource.BinarySI), // 80Mi
		},
	}

	// Large (1.5x small)
	sizeLargeResources = corev1.ResourceRequirements{
		Limits: corev1.ResourceList{
			"cpu":    resource.MustParse("750m"),
			"memory": *resource.NewQuantity(805306368, resource.BinarySI), // 768Mi
		},
		Requests: corev1.ResourceList{
			"cpu":    resource.MustParse("15m"),
			"memory": *resource.NewQuantity(100663296, resource.BinarySI), // 96Mi
		},
	}
)

func modifySizingConfig(c *common.OdigosConfiguration,
	clusterCollectorConfig common.CollectorGatewayConfiguration,
	nodeCollectorConfig common.CollectorNodeConfiguration) {

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
}

// hasResourceSettings checks if any resource setting is configured in the ResourceConfig
func hasResourceSettings(rc *common.ResourceConfig) bool {
	return rc.RequestMemoryMiB != 0 ||
		rc.LimitMemoryMiB != 0 ||
		rc.RequestCPUm != 0 ||
		rc.LimitCPUm != 0
}

// GetResourceRequirementsFromProfiles determines resource requirements based on the provided profiles
func GetResourceRequirementsFromProfiles(profileList []common.ProfileName) corev1.ResourceRequirements {
	if len(profileList) == 0 {
		return sizeSmallResources // Default to small if no profiles
	}

	// Track the highest priority profile (lower number = higher priority)
	// Default to small if no profiles are provided
	highestPriority := 999
	currentProfile := SizeSProfile.ProfileName
	for _, profile := range profileList {
		if priority, exists := SizeProfilePriority[profile]; exists && priority < highestPriority {
			highestPriority = priority
			currentProfile = profile
		}
	}

	// Return resources based on the highest priority profile found
	return ResourceRequirementsByProfile[currentProfile]
}
