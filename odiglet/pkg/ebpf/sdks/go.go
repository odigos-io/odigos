package sdks

import (
	"context"
	"errors"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"google.golang.org/grpc"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"

	"go.opentelemetry.io/auto"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

// log is the package logger for Go eBPF SDK; init once, use everywhere (respects ODIGOS_LOG_LEVEL).
var log = commonlogger.LoggerCompat().With("subsystem", "ebpfgosdk")

type GoOtelEbpfSdk struct {
	inst *auto.Instrumentation
	cp   *ebpf.ConfigProvider[auto.InstrumentationConfig]
}

// compile-time check that configProvider[auto.InstrumentationConfig] implements auto.Provider
var _ auto.ConfigProvider = (*ebpf.ConfigProvider[auto.InstrumentationConfig])(nil)

type GoInstrumentationFactory struct {
	otlpConn *grpc.ClientConn
}

func NewGoInstrumentationFactory(otlpConn *grpc.ClientConn) (instrumentation.Factory, error) {
	if otlpConn == nil {
		return nil, errors.New("otlp common connection can't be nil")
	}
	return &GoInstrumentationFactory{otlpConn: otlpConn}, nil
}

func (g *GoInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings instrumentation.Settings) (instrumentation.Instrumentation, error) {
	defaultExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithGRPCConn(g.otlpConn),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create exporter: %w", err)
	}

	initialConfig, err := convertToGoInstrumentationConfig(settings.InitialConfig)
	if err != nil {
		return nil, fmt.Errorf("invalid initial config type, expected *odigosv1.ContainerAgentConfig, got %T", settings.InitialConfig)
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
		log.Error("instrumentation setup failed", "err", err)
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

func convertToGoInstrumentationConfig(config instrumentation.Config) (auto.InstrumentationConfig, error) {
	containerConfig, ok := config.(*odigosv1.ContainerAgentConfig)
	if !ok {
		return auto.InstrumentationConfig{}, fmt.Errorf("invalid config type, expected *odigosv1.ContainerAgentConfig, got %T", config)
	}
	ic := auto.InstrumentationConfig{}
	if containerConfig == nil || containerConfig.Traces == nil {
		log.Info("No traces config provided for Go instrumentation, using default")
		return ic, nil
	}

	if containerConfig.Traces.TraceVerbosity != nil {
		ic.InstrumentationLibraryConfigs = make(map[auto.InstrumentationLibraryID]auto.InstrumentationLibrary)
		falseVal := false
		trueVal := true
		for _, lib := range containerConfig.Traces.TraceVerbosity.DisabledLibraries {
			libID := auto.InstrumentationLibraryID{InstrumentedPkg: lib.LibraryName}
			ic.InstrumentationLibraryConfigs[libID] = auto.InstrumentationLibrary{TracesEnabled: &falseVal}
		}
		for _, lib := range containerConfig.Traces.TraceVerbosity.EnabledLibraries {
			libID := auto.InstrumentationLibraryID{InstrumentedPkg: lib.LibraryName}
			ic.InstrumentationLibraryConfigs[libID] = auto.InstrumentationLibrary{TracesEnabled: &trueVal}
		}
	}

	// TODO: take sampling config from containerConfig.Traces.HeadSampling
	ic.Sampler = auto.DefaultSampler()
	return ic, nil
}
