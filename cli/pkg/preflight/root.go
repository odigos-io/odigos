package preflight

import (
	"context"

	"github.com/odigos-io/odigos/cli/pkg/kube"
)

type Check interface {
	Description() string
	Execute(client *kube.Client, ctx context.Context) error
}

var AllChecks = []Check{&isOdigosInstalled{}, &isDestinationConfigured{}}
