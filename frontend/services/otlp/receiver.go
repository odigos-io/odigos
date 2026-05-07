package otlp

import (
	"context"
	"fmt"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config/configgrpc"
	"go.opentelemetry.io/collector/config/confignet"
	"go.opentelemetry.io/collector/config/configoptional"
	"go.opentelemetry.io/collector/pdata/pcommon"
	xreceiver "go.opentelemetry.io/collector/receiver"
	"go.opentelemetry.io/collector/receiver/otlpreceiver"
	noopmetric "go.opentelemetry.io/otel/metric/noop"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// noExtensionsHost is a minimal component host for embedded receiver usage.
// It intentionally exposes no extensions.
type noExtensionsHost struct{}

func (h *noExtensionsHost) GetExtensions() map[component.ID]component.Component {
	return map[component.ID]component.Component{}
}

type Receiver struct {
	Factory  xreceiver.Factory
	Settings xreceiver.Settings

	Cfg *otlpreceiver.Config

	Host component.Host
	Port int
}

func NewReceiver(port int) (*Receiver, error) {
	f := otlpreceiver.NewFactory()

	cfg, ok := f.CreateDefaultConfig().(*otlpreceiver.Config)

	if !ok {
		return nil, fmt.Errorf("error parsing otlp otlpReceiver config")
	}

	grpcCfg := configgrpc.NewDefaultServerConfig()
	grpcCfg.NetAddr = confignet.AddrConfig{
		Endpoint:  fmt.Sprintf("0.0.0.0:%d", port),
		Transport: confignet.TransportTypeTCP,
	}

	// we only open gRPC port on 4317 and no http port
	cfg.GRPC = configoptional.Some(grpcCfg)
	cfg.HTTP = configoptional.None[otlpreceiver.HTTPConfig]()

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("validate receiver config: %w", err)
	}

	return &Receiver{
		Factory: f,
		Settings: xreceiver.Settings{
			ID: component.NewIDWithName(f.Type(), "odigos-ui"),
			TelemetrySettings: component.TelemetrySettings{
				Logger:         commonlogger.Logger().Named("otlp-receiver"),
				TracerProvider: nooptrace.NewTracerProvider(),
				MeterProvider:  noopmetric.NewMeterProvider(),
				Resource:       pcommon.NewResource(),
			},
			BuildInfo: component.NewDefaultBuildInfo(),
		},
		Cfg:  cfg,
		Host: &noExtensionsHost{},
		Port: port,
	}, nil
}

// Start registers every pipeline, then starts each.
func (r *Receiver) Start(ctx context.Context, pipelines ...OTLPPipeline) error {
	for _, p := range pipelines {
		if p == nil {
			continue
		}
		if err := p.Register(ctx); err != nil {
			return fmt.Errorf("otlp: register: %w", err)
		}
	}
	for _, p := range pipelines {
		if p == nil {
			continue
		}
		if err := p.Start(ctx); err != nil {
			return fmt.Errorf("otlp: start: %w", err)
		}
	}
	return nil
}

func (r *Receiver) WaitAndShutdown(ctx context.Context, pipelines ...OTLPPipeline) error {
	<-ctx.Done()
	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := len(pipelines) - 1; i >= 0; i-- {
		c := pipelines[i]
		if c == nil {
			continue
		}
		_ = c.Shutdown(shutCtx)
	}
	return nil
}
