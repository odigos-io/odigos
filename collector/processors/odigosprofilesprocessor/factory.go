package odigosprofilesprocessor

//go:generate mdatagen metadata.yaml

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"

	"github.com/odigos-io/odigos/collector/processors/odigosprofilesprocessor/internal/metadata"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// createDefaultConfig returns an empty Config; the YAML must set odigos_config_extension
// (Validate enforces this). Keeping the platform binding (extension type) out of the
// processor binary so the processor stays signal- and platform-neutral.
func createDefaultConfig() component.Config {
	return &Config{}
}

// NewFactory returns a profiles-only processor factory.
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
	proc := newOdigosProfilesProcessor(set.Logger, set.TelemetrySettings, oCfg)

	return xprocessorhelper.NewProfiles(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processProfiles,
		xprocessorhelper.WithCapabilities(processorCapabilities),
		xprocessorhelper.WithStart(proc.Start),
		xprocessorhelper.WithShutdown(proc.Shutdown),
	)
}
