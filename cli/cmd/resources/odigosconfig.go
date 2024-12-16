package resources

import (
	"context"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	k8sprofiles "github.com/odigos-io/odigos/k8sutils/pkg/profiles"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func NewOdigosConfiguration(ns string, config *common.OdigosConfiguration) (kube.Object, error) {
	data, err := yaml.Marshal(config)
	if err != nil {
		return nil, err
	}
	return &v1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      consts.OdigosConfigurationName,
			Namespace: ns,
		},
		Data: map[string]string{
			consts.OdigosConfigurationFileName: string(data),
		},
	}, nil
}

type odigosConfigResourceManager struct {
	client     *kube.Client
	ns         string
	config     *common.OdigosConfiguration
	odigosTier common.OdigosTier
}

func NewOdigosConfigResourceManager(client *kube.Client, ns string, config *common.OdigosConfiguration, odigosTier common.OdigosTier) resourcemanager.ResourceManager {
	return &odigosConfigResourceManager{client: client, ns: ns, config: config, odigosTier: odigosTier}
}

func (a *odigosConfigResourceManager) Name() string { return "OdigosConfig" }

func (a *odigosConfigResourceManager) InstallFromScratch(ctx context.Context) error {

	sizingProfile := k8sprofiles.FilterSizeProfiles(a.config.Profiles)

	collectorGatewayConfig := GetGatewayConfigBasedOnSize(sizingProfile)
	a.config.CollectorGateway = collectorGatewayConfig

	collectorNodeConfig := GetNodeCollectorConfigBasedOnSize(sizingProfile)
	if a.config.CollectorNode != nil && a.config.CollectorNode.CollectorOwnMetricsPort != 0 {
		if collectorNodeConfig == nil {
			collectorNodeConfig = &common.CollectorNodeConfiguration{}
		}
		collectorNodeConfig.CollectorOwnMetricsPort = a.config.CollectorNode.CollectorOwnMetricsPort
	}
	a.config.CollectorNode = collectorNodeConfig

	obj, err := NewOdigosConfiguration(a.ns, a.config)
	if err != nil {
		return err
	}

	resources := []kube.Object{
		obj,
	}
	return a.client.ApplyResources(ctx, a.config.ConfigVersion, resources)
}

func GetNodeCollectorConfigBasedOnSize(profile common.ProfileName) *common.CollectorNodeConfiguration {
	aggregateProfiles := append([]common.ProfileName{profile}, k8sprofiles.ProfilesMap[profile].Dependencies...)

	for _, profile := range aggregateProfiles {
		switch profile {
		case k8sprofiles.SizeSProfile.ProfileName:
			return &common.CollectorNodeConfiguration{
				RequestMemoryMiB: 150,
				LimitMemoryMiB:   300,
				RequestCPUm:      150,
				LimitCPUm:        300,
			}
		case k8sprofiles.SizeMProfile.ProfileName:
			return &common.CollectorNodeConfiguration{
				RequestMemoryMiB: 250,
				LimitMemoryMiB:   500,
				RequestCPUm:      250,
				LimitCPUm:        500,
			}
		case k8sprofiles.SizeLProfile.ProfileName:
			return &common.CollectorNodeConfiguration{
				RequestMemoryMiB: 500,
				LimitMemoryMiB:   750,
				RequestCPUm:      500,
				LimitCPUm:        750,
			}
		}
	}
	// Return nil if no matching profile is found.
	return nil
}

func GetGatewayConfigBasedOnSize(profile common.ProfileName) *common.CollectorGatewayConfiguration {
	aggregateProfiles := append([]common.ProfileName{profile}, k8sprofiles.ProfilesMap[profile].Dependencies...)

	for _, profile := range aggregateProfiles {
		switch profile {
		case k8sprofiles.SizeSProfile.ProfileName:
			return &common.CollectorGatewayConfiguration{
				MinReplicas:      1,
				MaxReplicas:      5,
				RequestCPUm:      150,
				LimitCPUm:        300,
				RequestMemoryMiB: 300,
				LimitMemoryMiB:   300,
			}
		case k8sprofiles.SizeMProfile.ProfileName:
			return &common.CollectorGatewayConfiguration{
				MinReplicas:      2,
				MaxReplicas:      8,
				RequestCPUm:      500,
				LimitCPUm:        1000,
				RequestMemoryMiB: 500,
				LimitMemoryMiB:   600,
			}
		case k8sprofiles.SizeLProfile.ProfileName:
			return &common.CollectorGatewayConfiguration{
				MinReplicas:      3,
				MaxReplicas:      12,
				RequestCPUm:      750,
				LimitCPUm:        1250,
				RequestMemoryMiB: 750,
				LimitMemoryMiB:   850,
			}
		}
	}
	// Return nil if no matching profile is found.
	return nil
}
