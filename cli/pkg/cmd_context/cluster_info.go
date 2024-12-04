package cmdcontext

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/util/version"

	"github.com/odigos-io/odigos/cli/pkg/autodetect"
	"github.com/odigos-io/odigos/cli/pkg/kube"
)

var (
	ErrCtxWithoutClusterDetails = errors.New("context does not contain cluster info")
)

type clusterDetailsContextKeyType int

const currentClusterDetailsKey clusterDetailsContextKeyType = iota

// ContextWithClusterDetails returns a copy of parent with ClusterDetails set as the current details.
func ContextWithClusterDetails(parent context.Context, clusterDetails *autodetect.ClusterDetails) context.Context {
	return context.WithValue(parent, currentClusterDetailsKey, clusterDetails)
}

// ClusterDetailsFromContextOrExit returns the current cluster details from ctx.
//
// If no details are currently set in ctx the program will exit with an error message.
func ClusterDetailsFromContextOrExit(ctx context.Context)  *autodetect.ClusterDetails {
	details, err := ClusterDetailsFromContext(ctx)
	if err != nil {
		kube.PrintClientErrorAndExit(err)
	}
	return details
}

// ClusterDetailsFromContext returns the current cluster details from ctx.
func ClusterDetailsFromContext(ctx context.Context) ( *autodetect.ClusterDetails, error) {
	if ctx == nil {
		return nil, ErrCtxIsNil
	}
	if details, ok := ctx.Value(currentClusterDetailsKey).( *autodetect.ClusterDetails); ok {
		return details, nil
	}
	return nil, ErrCtxWithoutClusterDetails
}

// ClusterKindFromContext returns the current cluster kind from ctx.
// If no kind is currently set in ctx, it returns autodetect.KindUnknown
func ClusterKindFromContext(ctx context.Context) autodetect.Kind {
	details, err := ClusterDetailsFromContext(ctx)
	if err != nil {
		return autodetect.KindUnknown
	}
	return details.Kind
}

// K8SVersionFromContext returns the current k8s version from ctx.
// If no version is currently set in ctx, it returns nil
func K8SVersionFromContext(ctx context.Context) *version.Version {
	details, err := ClusterDetailsFromContext(ctx)
	if err != nil {
		return nil
	}
	return details.K8SVersion
}