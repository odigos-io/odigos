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
	"gopkg.in/yaml.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

func DeleteOldOdigosSystemObjects(ctx context.Context, client *kube.Client, ns string, config *common.OdigosConfiguration) error {
	resources := kube.GetManagedResources(ns)
	for _, resource := range resources {
		l := log.Print(fmt.Sprintf("Syncing %s", resource.Resource.Resource))
		err := client.DeleteOldOdigosSystemObjects(ctx, resource, config.ConfigVersion)
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
	if err != nil {
		return nil, err
	}
	var odigosConfig common.OdigosConfiguration
	if err := yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), odigosConfig); err != nil {
		return nil, err
	}
	return &odigosConfig, nil
}
