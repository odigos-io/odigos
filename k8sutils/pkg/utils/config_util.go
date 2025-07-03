package utils

import (
	"context"
	"errors"

	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// error to indicate specifically that odigos effective config is not found.
// it can be used to differentiate and react specifically to this error.
// the effective config is reconciled in the scheduler, so it is possible to have a situation where the config is not found when odigos starts.
var ErrOdigosEffectiveConfigNotFound = errors.New("odigos effective config not found")

// GetCurrentOdigosConfiguration is a helper function to get the current odigos config using a controller-runtime client
func GetCurrentOdigosConfiguration(ctx context.Context, k8sClient client.Client) (common.OdigosConfiguration, error) {
	var configMap v1.ConfigMap
	var odigosConfiguration common.OdigosConfiguration
	odigosSystemNamespaceName := env.GetCurrentNamespace()
	if err := k8sClient.Get(ctx, types.NamespacedName{Namespace: odigosSystemNamespaceName, Name: consts.OdigosEffectiveConfigName},
		&configMap); err != nil {
		if apierrors.IsNotFound(err) {
			return odigosConfiguration, ErrOdigosEffectiveConfigNotFound
		} else {
			return odigosConfiguration, err
		}
	}
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration); err != nil {
		return odigosConfiguration, err
	}
	return odigosConfiguration, nil
}
