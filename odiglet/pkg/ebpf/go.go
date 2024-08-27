package ebpf

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/utils"

	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/consts"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"go.opentelemetry.io/auto"
	goAutoConfig "go.opentelemetry.io/auto/config"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
)

type GoOtelEbpfSdk struct {
	inst *auto.Instrumentation
	cp    *configProvider[goAutoConfig.InstrumentationConfig]
}

// compile-time check that configProvider[goAutoConfig.InstrumentationConfig] implements goAutoConfig.Provider
var _ goAutoConfig.Provider = (*configProvider[goAutoConfig.InstrumentationConfig])(nil)

type GoInstrumentationFactory struct{}

func NewGoInstrumentationFactory() InstrumentationFactory[*GoOtelEbpfSdk] {
	return &GoInstrumentationFactory{}
}

func (g *GoInstrumentationFactory) CreateEbpfInstrumentation(ctx context.Context, pid int, serviceName string, podWorkload *workload.PodWorkload, containerName string, podName string, loadedIndicator chan struct{}) (*GoOtelEbpfSdk, error) {
	defaultExporter, err := otlptracegrpc.New(
		ctx,
		otlptracegrpc.WithInsecure(),
		otlptracegrpc.WithEndpoint(fmt.Sprintf("%s:%d", env.Current.NodeIP, consts.OTLPPort)),
	)
	if err != nil {
		log.Logger.Error(err, "failed to create exporter")
		return nil, err
	}

	cp := newConfigProvider(goAutoConfig.InstrumentationConfig{})

	inst, err := auto.NewInstrumentation(
		ctx,
		auto.WithEnv(), // for OTEL_LOG_LEVEL
		auto.WithPID(pid),
		auto.WithResourceAttributes(utils.GetResourceAttributes(podWorkload, podName)...),
		auto.WithServiceName(serviceName),
		auto.WithTraceExporter(defaultExporter),
		auto.WithGlobal(),
		auto.WithLoadedIndicator(loadedIndicator),
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

func (g *GoOtelEbpfSdk) Close(ctx context.Context) error {
	return g.inst.Close()
}

func (g *GoOtelEbpfSdk) ApplyConfig(ctx context.Context, instConfig *v1alpha1.InstrumentationConfig) error {
	return g.cp.SendConfig(ctx, convertToGoInstrumentationConfig(instConfig))
}

func convertToGoInstrumentationConfig(instConfig *v1alpha1.InstrumentationConfig) goAutoConfig.InstrumentationConfig {
	ic := goAutoConfig.InstrumentationConfig{}
	ic.InstrumentationLibraryConfigs = make(map[goAutoConfig.InstrumentationLibraryID]goAutoConfig.InstrumentationLibrary)
	for _, sdkConfig := range instConfig.Spec.SdkConfigs {
		if sdkConfig.Language != common.GoProgrammingLanguage {
			continue
		}
		for _, ilc := range sdkConfig.InstrumentationLibraryConfigs {
			libID := goAutoConfig.InstrumentationLibraryID{
				InstrumentedPkg: ilc.InstrumentationLibraryId.InstrumentationLibraryName,
				SpanKind:        common.SpanKindOdigosToOtel(ilc.InstrumentationLibraryId.SpanKind),
			}
			var tracesEnabled *bool
			if ilc.TraceConfig != nil {
				tracesEnabled = ilc.TraceConfig.Enabled
			}
			ic.InstrumentationLibraryConfigs[libID] = goAutoConfig.InstrumentationLibrary{
				TracesEnabled: tracesEnabled,
			}
		}

		// TODO: sampling config
	}
	return ic
}
