package odigossqldboperationprocessor

import (
	"context"
	"strings"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

// NewFactory returns a new factory for the Resource processor.
func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("odigossqldboperationprocessor"),
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, component.StabilityLevelBeta),
	)
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func normalizeLanguages(languages []string) []string {
	normalizedLanguages := make([]string, len(languages))
	for i, language := range languages {
		normalizedLanguages[i] = strings.ToLower(strings.TrimSpace(language))
	}
	return normalizedLanguages
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces) (processor.Traces, error) {

	config := cfg.(*Config)

	// make sure the languages are normalized, (Java -> java)
	if config != nil && config.Exclude != nil {
		config.Exclude.Language = normalizeLanguages(config.Exclude.Language)
	}

	proc := &DBOperationProcessor{
		logger: set.Logger,
		config: config,
	}

	return processorhelper.NewTraces(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
