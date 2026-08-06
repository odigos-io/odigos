package tracecorrelations

import (
	"context"
	"errors"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/frontend/services"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ErrNotEnabled is returned when trace correlations are disabled in the effective config.
var ErrNotEnabled = errors.New("trace correlations are not enabled")

// IsEnabled reports whether service I/O trace correlations are enabled in effective-config.
func IsEnabled(ctx context.Context, c client.Client) (bool, error) {
	cfg, err := services.GetEffectiveConfig(ctx, c)
	if err != nil {
		return false, err
	}
	return ServiceIOEnabled(cfg), nil
}

// ServiceIOEnabled reports whether service I/O trace correlations are enabled in the given config.
func ServiceIOEnabled(cfg *common.OdigosConfiguration) bool {
	if cfg == nil {
		return false
	}
	return common.TraceCorrelationsServiceIOPipelineActive(cfg.TraceCorrelations)
}
