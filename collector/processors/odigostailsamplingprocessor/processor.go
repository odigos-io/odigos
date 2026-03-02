package odigostailsamplingprocessor

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos/collector/extension/odigosworkloadconfigextension"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/pdata/ptrace"
	"go.uber.org/zap"
)

type tailSamplingProcessor struct {
	logger                *zap.Logger
	config                *Config
	odigosConfigExtension *odigosworkloadconfigextension.OdigosWorkloadConfig
}

func (p *tailSamplingProcessor) processTraces(ctx context.Context, td ptrace.Traces) (ptrace.Traces, error) {
	resourceSpans := td.ResourceSpans()
	for i := 0; i < resourceSpans.Len(); i++ {
		resourceSpan := resourceSpans.At(i)
		scopeSpans := resourceSpan.ScopeSpans()
		for j := 0; j < scopeSpans.Len(); j++ {
			scopeSpan := scopeSpans.At(j)
			spans := scopeSpan.Spans()
			for k := 0; k < spans.Len(); k++ {
				span := spans.At(k)
				fmt.Println(span.Name())
			}
		}
	}
	return td, nil
}

func (p *tailSamplingProcessor) Start(ctx context.Context, host component.Host) error {
	// the extension name is validated as not nil in the config validate function
	// and can be nil in tests
	if p.config.OdigosConfigExtension != nil {
		ext, found := host.GetExtensions()[*p.config.OdigosConfigExtension]
		if !found || ext == nil {
			return fmt.Errorf("odigos config extension not found")
		}
		p.odigosConfigExtension = ext.(*odigosworkloadconfigextension.OdigosWorkloadConfig)
	}
	return nil
}

func newTailSamplingProcessor(logger *zap.Logger, cfg *Config) *tailSamplingProcessor {
	return &tailSamplingProcessor{
		logger: logger,
		config: cfg,
	}
}
