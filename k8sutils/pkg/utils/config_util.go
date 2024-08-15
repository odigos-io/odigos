package utils

import (
	"context"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func GetCurrentConfig(ctx context.Context, k8sClient client.Client) (common.OdigosConfiguration, error) {
	var configMap v1.ConfigMap
	var odigosConfig common.OdigosConfiguration
	odigosSystemNamespaceName := env.GetCurrentNamespace()
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: odigosSystemNamespaceName, Name: consts.OdigosConfigurationName}, &configMap); err != nil {
		return odigosConfig, err
	}
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfig); err != nil {
		return odigosConfig, err
	}
	return odigosConfig, nil
}
