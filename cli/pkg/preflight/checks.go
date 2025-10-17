package preflight

import (
	"context"
	"errors"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/remote"
)

type isOdigosInstalled struct{}

func (c *isOdigosInstalled) Description() string {
	return "Checking if Odigos is installed"
}

func (c *isOdigosInstalled) Execute(client *kube.Client, ctx context.Context, remote bool) error {
	_, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		if resources.IsErrNoOdigosNamespaceFound(err) {
			return errors.New("Odigos is NOT installed in the current cluster")
		} else {
			return fmt.Errorf("Error detecting Odigos namespace in the current cluster: %s", err)
		}
	}

	return nil
}

type isOdigosReady struct{}

func (c *isOdigosReady) Description() string {
	return "Checking if Odigos components are ready"
}

func (c *isOdigosReady) Execute(client *kube.Client, ctx context.Context, isRemote bool) error {
	ns, err := resources.GetOdigosNamespace(client, ctx)
	odigletDs, err := client.Clientset.AppsV1().DaemonSets(ns).Get(ctx, k8sconsts.OdigletDaemonSetName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Error getting Odigos odiglet daemonset: %s", err)
	}

	if odigletDs.Status.DesiredNumberScheduled != odigletDs.Status.NumberReady {
		return errors.New("Odigos odiglet daemonset is not ready")
	}

	instrumentor, err := client.Clientset.AppsV1().Deployments(ns).Get(ctx, k8sconsts.InstrumentorDeploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("Error getting Odigos instrumentor deployment: %s", err)
	}

	if instrumentor.Status.Replicas != instrumentor.Status.ReadyReplicas {
		return errors.New("Odigos instrumentor deployment is not ready")
	}

	return nil
}

type isDestinationConfigured struct{}

func (c *isDestinationConfigured) Description() string {
	return "Checking if at least one destination is configured"
}

func (c *isDestinationConfigured) Execute(client *kube.Client, ctx context.Context, isRemote bool) error {
	ns, err := resources.GetOdigosNamespace(client, ctx)
	if err != nil {
		return fmt.Errorf("Error detecting Odigos namespace in the current cluster: %s", err)
	}

	if isRemote {
		numDests, err := remote.GetNumberOfDestinations(ctx)
		if err != nil {
			return fmt.Errorf("Error listing Odigos destinations: %s", err)
		}

		if numDests == 0 {
			return errors.New("No Odigos destinations found")
		}

		return nil
	}

	destinations, err := client.OdigosClient.Destinations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Error listing Odigos destinations: %s", err)
	}
	if len(destinations.Items) == 0 {
		return errors.New("No Odigos destinations found")
	}

	return nil
}
