package resources

import (
	"context"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/cli/cmd/resources/resourcemanager"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ApplyResourceManagers(ctx context.Context, client *kube.Client, resourceManagers []resourcemanager.ResourceManager, prefixForLogging string) error {
	for _, rm := range resourceManagers {
		l := log.Print(fmt.Sprintf("%s %s", prefixForLogging, rm.Name()))
		err := rm.InstallFromScratch(ctx)
		if err != nil {
			l.Error(err)
			os.Exit(1)
		}
		l.Success()
	}
	return nil
}

func DeleteOldOdigosSystemObjects(ctx context.Context, client *kube.Client, ns string, config *v1alpha1.OdigosConfiguration) error {
	resources := kube.GetManagedResources(ns)
	for _, resource := range resources {
		l := log.Print(fmt.Sprintf("Syncing %s", resource.Resource.Resource))
		err := client.DeleteOldOdigosSystemObjects(ctx, resource, config.Spec.ConfigVersion)
		if err != nil {
			l.Error(err)
			os.Exit(1)
		}
		l.Success()
	}

	return nil
}

func GetCurrentConfig(ctx context.Context, client *kube.Client, ns string) (*v1alpha1.OdigosConfiguration, error) {
	return client.OdigosClient.OdigosConfigurations(ns).Get(ctx, OdigosConfigName, metav1.GetOptions{})
}
