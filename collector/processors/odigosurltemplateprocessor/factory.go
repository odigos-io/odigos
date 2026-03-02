package odigosurltemplateprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processor/odigosurltemplateprocessor/internal/metadata"
)

//go:generate mdatagen metadata.yaml

var consumerCapabilities = consumer.Capabilities{MutatesData: true}

// workloadRulesProvider is a local interface that the processor uses to obtain
// per-workload URL templatization rules from the workload config extension.
// It is satisfied by *odigosworkloadconfigextension.OdigosWorkloadConfig without
// importing that package directly.
type workloadRulesProvider interface {
	GetWorkloadUrlTemplatizationRules(attrs pcommon.Map) (rules []string, optedIn bool)
}

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

// extensionStartWrapper wraps a processor.Traces to inject the workload rules extension at Start() time.
// It locates the extension by component type, waits for its cache to sync, and then injects it as
// the processor's workloadRulesProvider.
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
		// Config.Validate() already checks this, so this should not happen.
		w.logger.Error("invalid workload config extension type", zap.String("type", extTypeStr), zap.Error(err))
	} else {
		for id, ext := range host.GetExtensions() {
			if id.Type() == extType {
				if provider, ok := ext.(workloadRulesProvider); ok {
					w.proc.provider = provider
					// Wait for the extension cache to sync before accepting spans.
					// WaitForCacheSync is asserted via a secondary interface to keep
					// this package decoupled from the extension package.
					type cacheSyncer interface {
						WaitForCacheSync(context.Context) bool
					}
					if syncer, ok := ext.(cacheSyncer); ok {
						if !syncer.WaitForCacheSync(ctx) {
							// C3: log warning but do not block startup
							w.logger.Warn("workload config extension cache sync did not complete; some spans may be missed on startup")
						}
					}
				} else {
					w.logger.Error("workload config extension does not implement workloadRulesProvider",
						zap.String("type", extTypeStr))
				}
				break
			}
		}
		if w.proc.provider == nil {
			w.logger.Warn("workload config extension not found; processor will apply heuristics to all spans",
				zap.String("type", extTypeStr))
		}
	}
	return w.inner.Start(ctx, host)
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
