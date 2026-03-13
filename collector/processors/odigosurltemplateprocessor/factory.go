package odigosurltemplateprocessor

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
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

	opts := []processorhelper.Option{processorhelper.WithCapabilities(consumerCapabilities)}
	if oCfg.WorkloadConfigExtensionID != "" {
		opts = append(opts, processorhelper.WithStart(func(ctx context.Context, host component.Host) error {
			return resolveAndRegisterExtension(ctx, host, proc, oCfg.WorkloadConfigExtensionID, set.Logger)
		}))
	}

	return processorhelper.NewTraces(ctx, set, cfg, nextConsumer, proc.processTraces, opts...)
}

// resolveAndRegisterExtension finds the OdigosConfigExtension by type (and optional named instance), registers the processor as callback, and waits for cache sync.
func resolveAndRegisterExtension(ctx context.Context, host component.Host, proc *urlTemplateProcessor, extTypeStr string, logger *zap.Logger) error {
	extType, err := component.NewType(extTypeStr)
	if err != nil {
		return fmt.Errorf("invalid workload config extension type %q: %w", extTypeStr, err)
	}
	extensions := host.GetExtensions()
	directID := component.NewID(extType)
	if ext, ok := extensions[directID]; ok {
		tryRegisterWithExtension(ext, proc, directID.String(), logger)
	} else {
		// Fallback when extension is registered with a named ID (e.g. odigos_config_k8s/production).
		for id, ext := range extensions {
			if id.Type() == extType {
				tryRegisterWithExtension(ext, proc, id.String(), logger)
				break
			}
		}
	}
	if proc.provider != nil {
		if !proc.provider.WaitForCacheSync(ctx) {
			logger.Warn("workload config extension cache sync did not complete; some spans may be missed on startup")
		}
	}
	if proc.provider == nil {
		logger.Info("workload config extension not found; using static rules from config",
			zap.String("type", extTypeStr))
	}
	return nil
}

func tryRegisterWithExtension(ext component.Component, proc *urlTemplateProcessor, extensionID string, logger *zap.Logger) {
	odigosExt, ok := ext.(collector.OdigosConfigExtension)
	if !ok {
		logger.Warn("extension does not implement OdigosConfigExtension", zap.String("extension_id", extensionID), zap.String("extGoType", fmt.Sprintf("%T", ext)))
		return
	}
	proc.provider = odigosExt
	odigosExt.RegisterWorkloadConfigCacheCallback(proc)
}
