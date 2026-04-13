package otlp

import (
	"context"
	"fmt"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/frontend/services/profiles"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	otlprecvfactory "go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

type Consumers struct {
	Metrics  *collectormetrics.OdigosMetricsConsumer
	Profiles *profiles.OdigosProfilesConsumer
}

func NewConsumers(metrics *collectormetrics.OdigosMetricsConsumer, profiles *profiles.OdigosProfilesConsumer) *Consumers {
	return &Consumers{Metrics: metrics, Profiles: profiles}
}

// Run creates and starts the OTLP receiver until ctx is canceled.
func (c *Consumers) Run(ctx context.Context, port int) {
	log := commonlogger.LoggerCompat().With("subsystem", "ui-otlp", "component", "otlp-receiver")

	f := otlprecvfactory.NewFactory()
	cfg, ok := f.CreateDefaultConfig().(*otlprecvfactory.Config)
	if !ok {
		panic("failed to cast default config to otlpreceiver.Config")
	}
	cfg.GRPC = configoptional.Some(configgrpc.ServerConfig{
		NetAddr: confignet.AddrConfig{
			Endpoint:  fmt.Sprintf("0.0.0.0:%d", port),
			Transport: confignet.TransportTypeTCP,
		},
	})

	host := componenttest.NewNopHost()
	set := receivertest.NewNopSettings(f.Type())

	// starting metrics receiver
	metricsRcv, err := f.CreateMetrics(ctx, set, cfg, c.Metrics)
	if err != nil {
		panic("failed to create OTLP metrics receiver")
	}
	if err := metricsRcv.Start(ctx, host); err != nil {
		log.Error("failed to start OTLP metrics receiver", "err", err)
	}

	// optionally starting profiling receiver
	var profilesRcv xreceiver.Profiles
	if c.Profiles != nil {
		if xf, xok := f.(xreceiver.Factory); !xok {
			log.Warn("OTLP receiver factory does not support profiles; continuing with metrics only")
		} else {
			profilesRcv, err = xf.CreateProfiles(ctx, set, cfg, c.Profiles.OTLPProfiles())
			if err != nil {
				log.Error("failed to create OTLP profiles receiver; continuing with metrics only", "err", err)
				profilesRcv = nil
			}
		}
	}
	if profilesRcv != nil {
		if err := profilesRcv.Start(ctx, host); err != nil {
			log.Error("failed to start OTLP profiles receiver", "err", err)
		}
	}
	if c.Profiles != nil && profilesRcv == nil {
		log.Warn("OTLP profiles receiver was not created; profiling ingestion disabled (gateway may log Unimplemented on ProfilesService)")
	}

	defer shutdownOTLPReceivers(metricsRcv, profilesRcv)

	log.Info("OTLP gRPC listening",
		"endpoint", fmt.Sprintf("0.0.0.0:%d", port),
		"metricsConsumer", true,
		"profilesConsumer", profilesRcv != nil,
	)
	<-ctx.Done()
}

func shutdownOTLPReceivers(metricsReceiver, profilesReceiver interface{ Shutdown(context.Context) error }) {
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if profilesReceiver != nil {
		_ = profilesReceiver.Shutdown(shutdownCtx)
	}
	_ = metricsReceiver.Shutdown(shutdownCtx)
}
