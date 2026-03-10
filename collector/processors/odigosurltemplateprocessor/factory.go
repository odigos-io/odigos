package odigosurltemplateprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processor/odigosurltemplateprocessor/internal/metadata"
	"github.com/odigos-io/odigos/common/collector"
)

//go:generate mdatagen metadata.yaml

var consumerCapabilities = consumer.Capabilities{MutatesData: true}

// NewFactory creates a new ProcessorFactory with default configuration
func NewFactory() processor.Factory {
	return processor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, metadata.TracesStability),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces,
) (processor.Traces, error) {
	oCfg := cfg.(*Config)
	proc, err := newUrlTemplateProcessor(set, oCfg)
	if err != nil {
		return nil, err
	}

	inner, err := processorhelper.NewTraces(ctx, set, cfg, nextConsumer, proc.processTraces, processorhelper.WithCapabilities(consumerCapabilities))
	if err != nil {
		return nil, err
	}

	if oCfg.WorkloadConfigExtensionID == "" {
		return inner, nil
	}

	return &extensionStartWrapper{
		inner:  inner,
		proc:   proc,
		cfg:    oCfg,
		logger: set.Logger,
	}, nil
}

// extensionStartWrapper wraps a processor.Traces to inject the OdigosConfigExtension at Start() time.
// It locates the extension by component type, waits for its cache to sync, and registers the processor as callback.
type extensionStartWrapper struct {
	inner  processor.Traces
	proc   *urlTemplateProcessor
	cfg    *Config
	logger *zap.Logger
}

func (w *extensionStartWrapper) Start(ctx context.Context, host component.Host) error {
	extTypeStr := w.cfg.WorkloadConfigExtensionID
	extType, err := component.NewType(extTypeStr)
	if err != nil {
		return fmt.Errorf("invalid workload config extension type %q: %w", extTypeStr, err)
	}
	extensions := host.GetExtensions()
	directID := component.NewID(extType)
	if ext, ok := extensions[directID]; ok {
		w.tryRegisterWithExtension(ext, directID.String())
	} else {
		for id, ext := range extensions {
			if id.Type() == extType {
				w.tryRegisterWithExtension(ext, id.String())
				break
			}
		}
	}
	if w.proc.provider != nil {
		if !w.proc.provider.WaitForCacheSync(ctx) {
			w.logger.Warn("workload config extension cache sync did not complete; some spans may be missed on startup")
		}
	}
	if w.proc.provider == nil {
		w.logger.Warn("workload config extension not found; processor will apply heuristics to all spans",
			zap.String("type", extTypeStr))
	}
	return w.inner.Start(ctx, host)
}

func (w *extensionStartWrapper) tryRegisterWithExtension(ext component.Component, extensionID string) {
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		w.logger.Warn("extension does not implement OdigosConfigExtension", zap.String("extension_id", extensionID), zap.String("extGoType", fmt.Sprintf("%T", ext)))
		return
	}
	w.proc.provider = odigosExt
	odigosExt.RegisterWorkloadConfigCacheCallback(w.proc)
}

func (w *extensionStartWrapper) Shutdown(ctx context.Context) error {
	return w.inner.Shutdown(ctx)
}

func (w *extensionStartWrapper) Capabilities() consumer.Capabilities {
	return w.inner.Capabilities()
}

func (w *extensionStartWrapper) ConsumeTraces(ctx context.Context, td ptrace.Traces) error {
	return w.inner.ConsumeTraces(ctx, td)
}
