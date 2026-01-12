package resources

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

func ApplyResourceManagers(ctx context.Context, client *kube.Client, resourceManagers []resourcemanager.ResourceManager, prefixForLogging string) error {
	for _, rm := range resourceManagers {
		l := log.Print(fmt.Sprintf("> %s %s", prefixForLogging, rm.Name()))
		err := rm.InstallFromScratch(ctx)
		if err != nil {
			l.Error(err)
			os.Exit(1)
		}
		l.Success()
	}
	return nil
}

func GetCurrentConfig(ctx context.Context, client *kube.Client, ns string) (*common.OdigosConfiguration, error) {
	configMap, err := client.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
	if err != nil && apierrors.IsNotFound(err) {
		// Fallback to the old config map name for backward compatibility
		configMap, err = client.CoreV1().ConfigMaps(ns).Get(ctx, consts.OdigosLegacyConfigName, metav1.GetOptions{})
		if err != nil {
			return nil, fmt.Errorf("failed to get odigos-config ConfigMap: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to get odigos-configuration ConfigMap: %w", err)
	}
	var odigosConfiguration common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration); err != nil {
		return nil, err
	}
	return &odigosConfiguration, nil
}
