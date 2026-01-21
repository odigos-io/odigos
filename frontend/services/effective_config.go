package services

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// GetEffectiveConfig retrieves the current effective configuration from the effective-config ConfigMap.
func GetEffectiveConfig(ctx context.Context) (*common.OdigosConfiguration, error) {
	ns := env.GetCurrentNamespace()

	cm, err := kube.DefaultClient.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosEffectiveConfigName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return &common.OdigosConfiguration{}, nil
		}
		return nil, fmt.Errorf("failed to get effective config: %w", err)
	}

	if cm.Data == nil || cm.Data[consts.OdigosConfigurationFileName] == "" {
		return &common.OdigosConfiguration{}, nil
	}

	var effectiveConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(cm.Data[consts.OdigosConfigurationFileName]), &effectiveConfig); err != nil {
		return nil, fmt.Errorf("failed to parse effective config: %w", err)
	}

	return &effectiveConfig, nil
}
