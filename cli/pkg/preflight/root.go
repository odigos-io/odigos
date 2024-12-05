package preflight

import (
	"context"

	"github.com/odigos-io/odigos/cli/pkg/kube"
)

type Check interface {
	Description() string
	Execute(client *kube.Client, ctx context.Context, remote bool) error
}

var AllChecks = []Check{&isOdigosInstalled{}, &isDestinationConfigured{}}
