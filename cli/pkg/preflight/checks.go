package preflight

import (
	"context"
	"errors"
	"fmt"

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
