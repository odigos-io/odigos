// Package ebpfprofilerwrapper wraps the upstream eBPF profiler receiver factory so the
// collector logger it uses downgrades benign, fail-safe load errors (e.g. an unsupported
// interpreter version) to warnings, via the shared common/logger rules. The receiver type,
// config and behavior are otherwise identical to the upstream receiver.
package ebpfprofilerwrapper

import (
	"context"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/xreceiver"
	profilercollector "go.opentelemetry.io/ebpf-profiler/collector"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewFactory() receiver.Factory {
	inner := profilercollector.NewFactory()
	xinner, ok := inner.(xreceiver.Factory)
	if !ok {
		return inner
	}
	return xreceiver.NewFactory(
		inner.Type(),
		inner.CreateDefaultConfig,
		xreceiver.WithProfiles(
			func(ctx context.Context, set receiver.Settings, cfg component.Config, next xconsumer.Profiles) (xreceiver.Profiles, error) {
				set.Logger = set.Logger.WithOptions(zap.WrapCore(func(c zapcore.Core) zapcore.Core {
					return commonlogger.NewDowngradeCore(c, commonlogger.DefaultDowngradeRules())
				}))
				return xinner.CreateProfiles(ctx, set, cfg, next)
			},
			xinner.ProfilesStability(),
		),
	)
}
