package odigossourcetodestinationfilterprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper"
)

func NewFactory() processor.Factory {
	return processor.NewFactory(
		component.MustNewType("odigossourcetodestinationfilterprocessor"),
		createDefaultConfig,
		processor.WithTraces(createTracesProcessor, component.StabilityLevelBeta),
	)
}

func createDefaultConfig() component.Config {
	return &Config{
		MatchConditions: []MatchCondition{},
	}
}

func createTracesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer consumer.Traces) (processor.Traces, error) {

	filterProc := &filterProcessor{
		logger: set.Logger,
		config: cfg.(*Config),
	}

	return processorhelper.NewTracesProcessor(
		ctx,
		set,
		cfg,
		nextConsumer,
		filterProc.processTraces,
		processorhelper.WithCapabilities(consumer.Capabilities{MutatesData: true}),
	)
}
