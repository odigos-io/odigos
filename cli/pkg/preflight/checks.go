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
	odigletDsList, err := client.Clientset.AppsV1().DaemonSets(ns).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", k8sconsts.OdigletDaemonSetName)})
	if err != nil {
		return fmt.Errorf("Error listing Odigos odiglet daemonsets: %s", err)
	}
	if len(odigletDsList.Items) == 0 {
		return errors.New("No Odigos odiglet daemonsets found")
	}
	if len(odigletDsList.Items) > 1 {
		return errors.New("Multiple Odigos odiglet daemonsets found")
	}
	odigletDs := &odigletDsList.Items[0]

	if odigletDs.Status.DesiredNumberScheduled != odigletDs.Status.NumberReady {
		return errors.New("Odigos odiglet daemonset is not ready")
	}

	instrumentors, err := client.Clientset.AppsV1().Deployments(ns).List(ctx, metav1.ListOptions{LabelSelector: fmt.Sprintf("app.kubernetes.io/name=%s", k8sconsts.InstrumentorDeploymentName)})
	if err != nil {
		return fmt.Errorf("Error listing Odigos instrumentor deployments: %s", err)
	}
	if len(instrumentors.Items) == 0 {
		return errors.New("No Odigos instrumentor deployments found")
	}
	if len(instrumentors.Items) > 1 {
		return errors.New("Multiple Odigos instrumentor deployments found")
	}
	instrumentor := instrumentors.Items[0]
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
