package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func getOdigosConfigFromConfigMap(ctx context.Context, c client.Client, configMapName string) (*common.OdigosConfiguration, error) {
	ns := env.GetCurrentNamespace()

	var cm v1.ConfigMap
	err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: configMapName}, &cm)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	if cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
		return nil, nil
	}

	var odigosConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
		return nil, fmt.Errorf("failed to parse odigos config: %w", err)
	}

	return &odigosConfig, nil
}

// GetEffectiveConfig retrieves the current effective configuration from the effective-config ConfigMap.
func GetEffectiveConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosEffectiveConfigName)
}

// GetHelmDeploymentConfig retrieves the current helm deployment configuration from the odigos-helm-deployment-config ConfigMap.
func GetHelmDeploymentConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosConfigurationName)
}

// GetRemoteConfig retrieves the current remote configuration from the odigos-remote-config ConfigMap.
func GetRemoteConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosRemoteConfigName)
}

// GetLocalUIConfig retrieves the current local UI configuration from the odigos-local-ui-config ConfigMap.
func GetLocalUIConfig(ctx context.Context, c client.Client) (*common.OdigosConfiguration, error) {
	return getOdigosConfigFromConfigMap(ctx, c, consts.OdigosLocalUiConfigName)
}

func PersistUiLocalSamplingConfig(ctx context.Context, c client.Client, samplingConfig *common.SamplingConfiguration) error {
	ns := env.GetCurrentNamespace()

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var cm v1.ConfigMap
		err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm)
		if err != nil {
			return err
		}
		config := common.OdigosConfiguration{}
		if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &config); err != nil {
			return err
		}
		config.Sampling = samplingConfig
		yamlText, err := yaml.Marshal(config)
		if err != nil {
			return err
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(yamlText)
		return c.Update(ctx, &cm)
	})
}
