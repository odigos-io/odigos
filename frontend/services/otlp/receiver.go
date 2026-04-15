package otlp

import (
	"context"
	"fmt"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	recv "go.opentelemetry.io/collector/receiver"
	otlprecvfactory "go.opentelemetry.io/collector/receiver/otlpreceiver"
	"go.opentelemetry.io/collector/receiver/receivertest"
	"go.opentelemetry.io/collector/receiver/xreceiver"
)

type Receiver struct {
	Factory  recv.Factory
	Cfg      *otlprecvfactory.Config
	Host     component.Host
	Settings recv.Settings
	Port     int

	metricsReceiver  recv.Metrics
	profilesReceiver xreceiver.Profiles
}

func NewReceiver(port int) (*Receiver, error) {
	f := otlprecvfactory.NewFactory()

	// Derive the default OTLP gRPC port config
	cfg, ok := f.CreateDefaultConfig().(*otlprecvfactory.Config)

	if !ok {
		return nil, fmt.Errorf("otlp: default config is not *otlpreceiver.Config")
	}

	grpcCfg := configgrpc.NewDefaultServerConfig()
	grpcCfg.NetAddr = confignet.AddrConfig{
		Endpoint:  fmt.Sprintf("0.0.0.0:%d", port),
		Transport: confignet.TransportTypeTCP,
	}

	cfg.GRPC = configoptional.Some(grpcCfg)
	cfg.HTTP = configoptional.None[otlprecvfactory.HTTPConfig]()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("otlp: validate receiver config: %w", err)
	}

	return &Receiver{
		Factory:  f,
		Cfg:      cfg,
		Host:     componenttest.NewNopHost(),
		Settings: receivertest.NewNopSettings(f.Type()),
		Port:     port,
	}, nil
}

// Setup registers every consumer, then starts each.
func (r *Receiver) Setup(ctx context.Context, consumers ...Consumer) error {
	for _, c := range consumers {
		if c == nil {
			continue
		}
		if err := c.Register(ctx, r); err != nil {
			return fmt.Errorf("otlp: register: %w", err)
		}
	}
	for _, c := range consumers {
		if c == nil {
			continue
		}
		if err := c.Start(ctx); err != nil {
			return fmt.Errorf("otlp: start: %w", err)
		}
	}
	return nil
}

func (r *Receiver) WaitAndShutdown(ctx context.Context, consumers ...Consumer) error {
	commonlogger.LoggerCompat().With("subsystem", "ui-otlp", "receiver").Info("OTLP gRPC running",
		"endpoint", fmt.Sprintf("0.0.0.0:%d", r.Port),
		"metrics", r.metricsReceiver != nil,
		"profiles", r.profilesReceiver != nil,
	)
	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := len(consumers) - 1; i >= 0; i-- {
		c := consumers[i]
		if c == nil {
			continue
		}
		_ = c.Shutdown(shutCtx)
	}
	return nil
}
