package sdks

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"

	"go.opentelemetry.io/auto"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

type GoOtelEbpfSdk struct {
	inst *auto.Instrumentation
	cp   *ebpf.ConfigProvider[auto.InstrumentationConfig]
}

// compile-time check that configProvider[auto.InstrumentationConfig] implements auto.Provider
var _ auto.ConfigProvider = (*ebpf.ConfigProvider[auto.InstrumentationConfig])(nil)

type GoInstrumentationFactory struct {
}

func NewGoInstrumentationFactory() instrumentation.Factory {
	return &GoInstrumentationFactory{}
}

func (g *GoInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings instrumentation.Settings) (instrumentation.Instrumentation, error) {
	defaultExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("localhost:%d", consts.OTLPPort)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	initialConfig, err := convertToGoInstrumentationConfig(settings.InitialConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid initial config type, expected *odigosv1.SdkConfig, got %T", settings.InitialConfig)
	}

	cp := ebpf.NewConfigProvider(initialConfig)

	inst, err := auto.NewInstrumentation(
		ctx,
		auto.WithEnv(), // for OTEL_LOG_LEVEL
		auto.WithPID(pid),
		auto.WithResourceAttributes(settings.ResourceAttributes...),
		auto.WithServiceName(settings.ServiceName),
		auto.WithTraceExporter(defaultExporter),
		auto.WithGlobal(),
		auto.WithConfigProvider(cp),
	)
	if err != nil {
		commonlogger.Logger().Error("instrumentation setup failed", "err", err)
		return nil, err
	}

	return &GoOtelEbpfSdk{inst: inst, cp: cp}, nil
}

func (g *GoOtelEbpfSdk) Run(ctx context.Context) error {
	return g.inst.Run(ctx)
}

func (g *GoOtelEbpfSdk) Load(ctx context.Context) (instrumentation.Status, error) {
	loadErr := g.inst.Load(ctx)
	return instrumentation.Status{}, loadErr
}

func (g *GoOtelEbpfSdk) Close(_ context.Context) error {
	return g.inst.Close()
}

func (g *GoOtelEbpfSdk) ApplyConfig(ctx context.Context, sdkConfig instrumentation.Config) error {
	updatedConfig, err := convertToGoInstrumentationConfig(sdkConfig)
	if err != nil {
		return err
	}

	return g.cp.SendConfig(ctx, updatedConfig)
}

func convertToGoInstrumentationConfig(sdkConfig instrumentation.Config) (auto.InstrumentationConfig, error) {
	initialConfig, ok := sdkConfig.(*odigosv1.SdkConfig)
	if !ok {
		return auto.InstrumentationConfig{}, fmt.Errorf("invalid initial config type, expected *odigosv1.SdkConfig, got %T", sdkConfig)
	}
	ic := auto.InstrumentationConfig{}
	if sdkConfig == nil {
		commonlogger.Logger().Info("No SDK config provided for Go instrumentation, using default")
		return ic, nil
	}
	ic.InstrumentationLibraryConfigs = make(map[auto.InstrumentationLibraryID]auto.InstrumentationLibrary)
	for _, ilc := range initialConfig.InstrumentationLibraryConfigs {
		libID := auto.InstrumentationLibraryID{
			InstrumentedPkg: ilc.InstrumentationLibraryId.InstrumentationLibraryName,
			SpanKind:        common.SpanKindOdigosToOtel(ilc.InstrumentationLibraryId.SpanKind),
		}
		var tracesEnabled *bool
		if ilc.TraceConfig != nil {
			tracesEnabled = ilc.TraceConfig.Enabled
		}
		ic.InstrumentationLibraryConfigs[libID] = auto.InstrumentationLibrary{
			TracesEnabled: tracesEnabled,
		}
	}

	// TODO: take sampling config from the CR
	ic.Sampler = auto.DefaultSampler()
	return ic, nil
}
