package cmdutil

import (
	"context"

	"github.com/odigos-io/odigos/cli/pkg/kube"
	"github.com/odigos-io/odigos/cli/pkg/log"
)

type ResourceCreationFunc func(ctx context.Context, client *kube.Client, ns string, labelKey string) error

func CreateKubeResourceWithLogging(ctx context.Context, msg string, client *kube.Client, ns string, labelKey string, create ResourceCreationFunc) {
	l := log.Print(msg)
	err := create(ctx, client, ns, labelKey)
	if err != nil {
		l.Error(err)
	}

	l.Success()
}
