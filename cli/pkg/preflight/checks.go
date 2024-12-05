package preflight

import (
	"context"
	"errors"
	"fmt"

	"github.com/odigos-io/odigos/cli/pkg/remote"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/odigos-io/odigos/cli/cmd/resources"
	"github.com/odigos-io/odigos/cli/pkg/kube"
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
		numDests, err := remote.GetNumberOfDestinations(ctx, client)
		if err != nil {
			return fmt.Errorf("Error listing Odigos destinations: %s", err)
		}

		if numDests == 0 {
			return errors.New("No Odigos destinations found")
		}

		return nil
	}

	dests, err := client.OdigosClient.Destinations(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("Error listing Odigos destinations: %s", err)
	}

	if len(dests.Items) == 0 {
		return errors.New("No Odigos destinations found")
	}

	return nil
}
