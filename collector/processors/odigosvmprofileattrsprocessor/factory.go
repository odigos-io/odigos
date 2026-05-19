package odigosvmprofileattrsprocessor

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"
)

var processorType = component.MustNewType("odigosvmprofileattrsprocessor")

// NewFactory returns a factory for the VM profile resource attributes processor.
func NewFactory() xprocessor.Factory {
	return xprocessor.NewFactory(
		processorType,
		func() component.Config { return createDefaultConfig() },
		xprocessor.WithProfiles(createProfilesProcessor, component.StabilityLevelBeta),
	)
}

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

	proc := &vmProfileAttrsProcessor{
		logger:    set.Logger,
		cfg:       oCfg,
		attrCache: newProfileAttrCache(),
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
