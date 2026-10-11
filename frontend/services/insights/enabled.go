// Package insights provides the frontend's access layer to the odigos-insights
// service (enterprise security feature). The service is optional: it is only
// deployed when insights are enabled in the Odigos configuration, so every
// operation in this package is gated on IsEnabled.
package insights

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/services"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrNotEnabled is returned by insights resolvers when the feature is not
// enabled in the effective configuration. Callers should match it with
// errors.Is rather than comparing error strings.
var ErrNotEnabled = errors.New("insights are not enabled")

// IsEnabled reports whether insights are enabled in the effective
// configuration, which is what determines if the odigos-insights service is
// deployed in the cluster.
func IsEnabled(ctx context.Context, c client.Client) (bool, error) {
	cfg, err := services.GetEffectiveConfig(ctx, c)
	if err != nil {
		return false, err
	}
	return EnabledFromOdigosConfig(cfg), nil
}

// EnabledFromOdigosConfig reports whether insights are enabled for an already
// loaded config snapshot, for callers that hold one and want to avoid a second
// read.
func EnabledFromOdigosConfig(cfg *common.OdigosConfiguration) bool {
	return cfg.InsightsEnabled()
}
