// factory.go wires the processor into the collector: a profiles-only factory
// that builds the symbolizeProcessor and registers its Shutdown hook.
package odigossymbolizeprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"

	"github.com/odigos-io/odigos/collector/processors/odigossymbolizeprocessor/internal/metadata"
)

// processorCapabilities is MutatesData: true because symbolization enriches the
// profile dictionary in place (appends Function/String entries, sets Location
// Lines).
var processorCapabilities = consumer.Capabilities{MutatesData: true}

func createDefaultConfig() component.Config {
	return &Config{}
}

// NewFactory returns a profiles-only native-symbolization processor factory.
func NewFactory() xprocessor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		createDefaultConfig,
		xprocessor.WithProfiles(createProfilesProcessor, metadata.ProfilesStability),
	)
}

func createProfilesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xprocessor.Profiles, error) {
	oCfg := cfg.(*Config)
	proc := newProcessor(set.Logger, oCfg)

	return xprocessorhelper.NewProfiles(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processProfiles,
		xprocessorhelper.WithCapabilities(processorCapabilities),
		xprocessorhelper.WithShutdown(proc.Shutdown),
	)
}
