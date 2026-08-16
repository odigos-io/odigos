package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// UpdateRemoteConfig updates the remote configuration in the odigos-remote-config ConfigMap.
func UpdateRemoteConfig(ctx context.Context, config *common.OdigosConfiguration) (*common.OdigosConfiguration, error) {
	ns := env.GetCurrentNamespace()

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosRemoteConfigName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			yamlBytes, err := yaml.Marshal(config)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal remote config: %w", err)
			}
			// Fetch odigos-configuration to use as owner reference.
			// This ensures odigos-remote-config is automatically deleted by Kubernetes GC
			// when odigos-configuration is deleted (e.g., during Helm uninstall).
			ownerCm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
			if err != nil {
				return nil, fmt.Errorf("failed to get odigos-configuration for owner reference: %w", err)
			}
			newCm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      consts.OdigosRemoteConfigName,
					Namespace: ns,
					Labels:    map[string]string{k8sconsts.OdigosSystemConfigLabelKey: "remote"},
					OwnerReferences: []metav1.OwnerReference{{
						APIVersion: "v1", Kind: "ConfigMap", Name: ownerCm.Name, UID: ownerCm.UID,
					}},
				},
				Data: map[string]string{consts.OdigosConfigurationFileName: string(yamlBytes)},
			}
			_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Create(ctx, newCm, metav1.CreateOptions{})
			return config, err
		}
		return nil, fmt.Errorf("failed to get remote config: %w", err)
	}

	currentConfig, err := OdigosConfigurationFromConfigMap(cm)
	if err != nil {
		return nil, fmt.Errorf("failed to parse remote config: %w", err)
	}
	if currentConfig == nil {
		currentConfig = &common.OdigosConfiguration{}
	}
	mergeRemoteConfigUpdate(currentConfig, config)

	yamlBytes, err := yaml.Marshal(currentConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal remote config: %w", err)
	}
	if cm.Data == nil {
		cm.Data = make(map[string]string)
	}
	cm.Data[consts.OdigosConfigurationFileName] = string(yamlBytes)

	_, err = kube.DefaultClient.CoreV1().ConfigMaps(ns).Update(ctx, cm, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update remote config ConfigMap: %w", err)
	}

	return currentConfig, nil
}

func mergeRemoteConfigUpdate(current, update *common.OdigosConfiguration) {
	if update == nil {
		return
	}
	if update.Rollout != nil {
		if current.Rollout == nil {
			current.Rollout = &common.RolloutConfiguration{}
		}
		current.Rollout.AutomaticRolloutDisabled = update.Rollout.AutomaticRolloutDisabled
		if update.Rollout.MaxConcurrentRollouts != 0 {
			current.Rollout.MaxConcurrentRollouts = update.Rollout.MaxConcurrentRollouts
		}
	}
	if update.Profiling == nil {
		return
	}
	if current.Profiling == nil {
		current.Profiling = &common.ProfilingConfiguration{}
	}

	src, dst := update.Profiling, current.Profiling
	if src.Enabled != nil {
		dst.Enabled = src.Enabled
	}
	if src.Exporter != nil {
		dst.Exporter = src.Exporter
	}
	if src.Symbolization != nil {
		dst.Symbolization = src.Symbolization
	}
	if src.Ui == nil {
		return
	}
	if dst.Ui == nil {
		dst.Ui = &common.ProfilingUiConfiguration{}
	}
	if src.Ui.MaxSlots != 0 {
		dst.Ui.MaxSlots = src.Ui.MaxSlots
	}
	if src.Ui.SlotMaxBytes != 0 {
		dst.Ui.SlotMaxBytes = src.Ui.SlotMaxBytes
	}
	if src.Ui.SlotTTLSeconds != 0 {
		dst.Ui.SlotTTLSeconds = src.Ui.SlotTTLSeconds
	}
}
