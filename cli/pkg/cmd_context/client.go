package cmdcontext

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/cli/pkg/kube"
)

var (
	ErrCtxIsNil = errors.New("context is nil when trying to get kube client")
	ErrCtxWithoutKubeClient = errors.New("context does not contain kube client")
)

type kubeClientContextKeyType int

const currentClientKey kubeClientContextKeyType = iota

// ContextWithKubeClient returns a copy of parent with kubeClient set as the current client.
func ContextWithKubeClient(parent context.Context, kubeClient *kube.Client) context.Context {
	return context.WithValue(parent, currentClientKey, kubeClient)
}

// KubeClientFromContextOrExit returns the current kube client from ctx.
//
// If no client is currently set in ctx the program will exit with an error message.
func KubeClientFromContextOrExit(ctx context.Context) *kube.Client {
	client, err := KubeClientFromContext(ctx)
	if err != nil {
		kube.PrintClientErrorAndExit(err)
	}
	return client
}

// KubeClientFromContext returns the current kube client from ctx.
func KubeClientFromContext(ctx context.Context) (*kube.Client, error) {
	if ctx == nil {
		return nil, ErrCtxIsNil
	}
	if client, ok := ctx.Value(currentClientKey).(*kube.Client); ok {
		return client, nil
	}
	return nil, ErrCtxWithoutKubeClient
}