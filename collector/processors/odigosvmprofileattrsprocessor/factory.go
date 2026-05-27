package odigosvmprofileattrsprocessor

//go:generate mdatagen metadata.yaml

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"

	"github.com/odigos-io/odigos/collector/processors/odigosvmprofileattrsprocessor/internal/metadata"
)

// NewFactory returns a factory for the VM profile resource attributes processor.
func NewFactory() xprocessor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		func() component.Config { return createDefaultConfig() },
		xprocessor.WithProfiles(createProfilesProcessor, metadata.ProfilesStability),
	)
}

// createProfilesProcessor builds the processor with its telemetry builder and chains the
// drop-empty consumer in front of the next exporter so empty batches never reach the wire.
func createProfilesProcessor(
	ctx context.Context,
	set processor.Settings,
	cfg component.Config,
	nextConsumer xconsumer.Profiles,
) (xprocessor.Profiles, error) {
	oCfg := cfg.(*Config)
	if err := oCfg.Validate(); err != nil {
		return nil, err
	}

	telemetryBuilder, err := metadata.NewTelemetryBuilder(set.TelemetrySettings)
	if err != nil {
		return nil, err
	}

	proc := &vmProfileAttrsProcessor{
		logger:           set.Logger,
		cfg:              oCfg,
		attrCache:        newProfileAttrCache(),
		telemetryBuilder: telemetryBuilder,
	}

	return xprocessorhelper.NewProfiles(
		ctx,
		set,
		cfg,
		&dropEmptyProfilesConsumer{
			next:   nextConsumer,
			logger: set.Logger,
		},
		proc.processProfiles,
		xprocessorhelper.WithCapabilities(proc.capabilities()),
		xprocessorhelper.WithStart(proc.start),
		xprocessorhelper.WithShutdown(proc.shutdown),
	)
}
