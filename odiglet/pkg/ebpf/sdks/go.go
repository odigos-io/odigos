package sdks

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"

	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
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

func NewGoInstrumentationFactory() ebpf.Factory {
	return &GoInstrumentationFactory{}
}

func (g *GoInstrumentationFactory) CreateInstrumentation(ctx context.Context, pid int, settings ebpf.Settings) (ebpf.Instrumentation, error) {
	defaultExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", env.Current.NodeIP, consts.OTLPPort)),
	)
	if err != nil {
		log.Logger.Error(err, "failed to create exporter")
		return nil, err
	}

	cp := ebpf.NewConfigProvider(convertToGoInstrumentationConfig(settings.InitialConfig))

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
		log.Logger.Error(err, "instrumentation setup failed")
		return nil, err
	}

	return &GoOtelEbpfSdk{inst: inst, cp: cp}, nil
}

func (g *GoOtelEbpfSdk) Run(ctx context.Context) error {
	return g.inst.Run(ctx)
}

func (g *GoOtelEbpfSdk) Load(ctx context.Context) error {
	return g.inst.Load(ctx)
}

func (g *GoOtelEbpfSdk) Close(_ context.Context) error {
	return g.inst.Close()
}

func (g *GoOtelEbpfSdk) ApplyConfig(ctx context.Context, sdkConfig *odigosv1.SdkConfig) error {
	return g.cp.SendConfig(ctx, convertToGoInstrumentationConfig(sdkConfig))
}

func convertToGoInstrumentationConfig(sdkConfig *odigosv1.SdkConfig) auto.InstrumentationConfig {
	ic := auto.InstrumentationConfig{}
	ic.InstrumentationLibraryConfigs = make(map[auto.InstrumentationLibraryID]auto.InstrumentationLibrary)
	for _, ilc := range sdkConfig.InstrumentationLibraryConfigs {
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
	return ic
}
