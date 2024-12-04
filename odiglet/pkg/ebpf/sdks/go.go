package sdks

import (
	"context"
	"fmt"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/kube/utils"
	"sigs.k8s.io/controller-runtime/pkg/client"

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

// compile-time check that GoOtelEbpfSdk implements ConfigurableOtelEbpfSdk
var _ ebpf.ConfigurableOtelEbpfSdk = (*GoOtelEbpfSdk)(nil)

type GoInstrumentationFactory struct{
	kubeclient client.Client
}

func NewGoInstrumentationFactory(kubeclient client.Client) ebpf.InstrumentationFactory[*GoOtelEbpfSdk] {
	return &GoInstrumentationFactory{
		kubeclient: kubeclient,
	}
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

	// Fetch initial config based on the InstrumentationConfig CR
	instrumentationConfig := &odigosv1.InstrumentationConfig{}
	initialConfig := auto.InstrumentationConfig{}
	instrumentationConfigKey := client.ObjectKey{
		Namespace: podWorkload.Namespace,
		Name:      workload.CalculateWorkloadRuntimeObjectName(podWorkload.Name, podWorkload.Kind),
	}
	if err := g.kubeclient.Get(ctx, instrumentationConfigKey, instrumentationConfig); err == nil {
		initialConfig = convertToGoInstrumentationConfig(instrumentationConfig)
	}

	cp := ebpf.NewConfigProvider(initialConfig)

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

func (g *GoOtelEbpfSdk) ApplyConfig(ctx context.Context, instConfig *odigosv1.InstrumentationConfig) error {
	return g.cp.SendConfig(ctx, convertToGoInstrumentationConfig(instConfig))
}

func convertToGoInstrumentationConfig(instConfig *odigosv1.InstrumentationConfig) auto.InstrumentationConfig {
	ic := auto.InstrumentationConfig{}
	ic.InstrumentationLibraryConfigs = make(map[auto.InstrumentationLibraryID]auto.InstrumentationLibrary)
	for _, sdkConfig := range instConfig.Spec.SdkConfigs {
		if sdkConfig.Language != common.GoProgrammingLanguage {
			continue
		}
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

		// TODO: sampling config
	}

	// TODO: take sampling config from the CR
	ic.Sampler = auto.DefaultSampler()
	return ic
}
