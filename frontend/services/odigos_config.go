package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

// upsertLocalUiConfig applies a mutation to the odigos-local-ui-config ConfigMap,
// creating it with proper owner references if it does not yet exist.
func upsertLocalUiConfig(ctx context.Context, c client.Client, mutate func(cfg *common.OdigosConfiguration)) error {
	ns := env.GetCurrentNamespace()

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		var cm v1.ConfigMap
		if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosLocalUiConfigName}, &cm); err != nil {
			if apierrors.IsNotFound(err) {
				ownerCm := v1.ConfigMap{}
				if err := c.Get(ctx, types.NamespacedName{Namespace: ns, Name: consts.OdigosConfigurationName}, &ownerCm); err != nil {
					return fmt.Errorf("failed to get odigos-configuration for owner reference: %w", err)
				}
				cfg := common.OdigosConfiguration{}
				mutate(&cfg)
				data, marshalErr := yaml.Marshal(cfg)
				if marshalErr != nil {
					return marshalErr
				}
				newCm := v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      consts.OdigosLocalUiConfigName,
						Namespace: ns,
						Labels:    map[string]string{k8sconsts.OdigosSystemConfigLabelKey: "local-ui"},
						OwnerReferences: []metav1.OwnerReference{{
							APIVersion: "v1", Kind: "ConfigMap", Name: ownerCm.Name, UID: ownerCm.UID,
						}},
					},
					Data: map[string]string{consts.OdigosConfigurationFileName: string(data)},
				}
				return c.Create(ctx, &newCm)
			}
			return err
		}

		var cfg common.OdigosConfiguration
		if cm.Data != nil && cm.Data[consts.OdigosConfigurationFileName] != "" {
			if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg); err != nil {
				return fmt.Errorf("parse existing config: %w", err)
			}
		}
		mutate(&cfg)
		data, err := yaml.Marshal(cfg)
		if err != nil {
			return err
		}
		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(data)
		return c.Update(ctx, &cm)
	})
}

func PersistUiLocalSamplingConfig(ctx context.Context, c client.Client, samplingConfig *common.SamplingConfiguration) error {
	return upsertLocalUiConfig(ctx, c, func(cfg *common.OdigosConfiguration) {
		cfg.Sampling = samplingConfig
	})
}
