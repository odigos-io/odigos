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
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/yaml"
)

// UpdateRemoteConfig applies a mutation to the config document in the odigos-remote-config
// ConfigMap, creating it with an owner reference if it does not exist yet.
//
// The document has several independent writers (the central backend and the UI own
// different fields) and the scheduler merges it as a whole over the helm baseline, so it
// is read-modify-written: a caller that manages only a subset of the fields must not drop
// the fields owned by another writer.
func UpdateRemoteConfig(ctx context.Context, mutate func(cfg *common.OdigosConfiguration)) error {
	ns := env.GetCurrentNamespace()
	configMaps := kube.DefaultClient.CoreV1().ConfigMaps(ns)

	return retry.RetryOnConflict(retry.DefaultBackoff, func() error {
		cm, err := configMaps.Get(ctx, consts.OdigosRemoteConfigName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return fmt.Errorf("failed to get remote config: %w", err)
			}

			// Fetch odigos-configuration to use as owner reference.
			// This ensures odigos-remote-config is automatically deleted by Kubernetes GC
			// when odigos-configuration is deleted (e.g., during Helm uninstall).
			ownerCm, ownerErr := configMaps.Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
			if ownerErr != nil {
				return fmt.Errorf("failed to get odigos-configuration for owner reference: %w", ownerErr)
			}
			cfg := common.OdigosConfiguration{}
			mutate(&cfg)
			yamlBytes, marshalErr := yaml.Marshal(cfg)
			if marshalErr != nil {
				return fmt.Errorf("failed to marshal remote config: %w", marshalErr)
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
			_, err = configMaps.Create(ctx, newCm, metav1.CreateOptions{})
			return err
		}

		cfg := common.OdigosConfiguration{}
		if cm.Data != nil && cm.Data[consts.OdigosConfigurationFileName] != "" {
			if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &cfg); err != nil {
				return fmt.Errorf("failed to parse existing remote config: %w", err)
			}
		}
		mutate(&cfg)
		yamlBytes, err := yaml.Marshal(cfg)
		if err != nil {
			return fmt.Errorf("failed to marshal remote config: %w", err)
		}

		if cm.Data == nil {
			cm.Data = make(map[string]string)
		}
		cm.Data[consts.OdigosConfigurationFileName] = string(yamlBytes)

		if _, err := configMaps.Update(ctx, cm, metav1.UpdateOptions{}); err != nil {
			return fmt.Errorf("failed to update remote config ConfigMap: %w", err)
		}
		return nil
	})
}
