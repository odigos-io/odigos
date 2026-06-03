package odigosnativesymbolizeprocessor

//go:generate mdatagen metadata.yaml

import (
	"context"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer/xconsumer"
	"go.opentelemetry.io/collector/processor"
	"go.opentelemetry.io/collector/processor/processorhelper/xprocessorhelper"
	"go.opentelemetry.io/collector/processor/xprocessor"

	"github.com/odigos-io/odigos/collector/processors/odigosnativesymbolizeprocessor/internal/metadata"
	"github.com/odigos-io/odigos/collector/processors/odigosnativesymbolizeprocessor/internal/symbolize"
)

// NewFactory returns a factory for the native symbolize processor.
func NewFactory() xprocessor.Factory {
	return xprocessor.NewFactory(
		metadata.Type,
		func() component.Config { return createDefaultConfig() },
		xprocessor.WithProfiles(createProfilesProcessor, metadata.ProfilesStability),
	)
}

// createProfilesProcessor builds the profiles processor that symbolizes native frames
// in-place and forwards every batch to the next consumer.
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

	proc := &nativeSymbolizeProcessor{
		logger: set.Logger,
		cfg:    oCfg,
		sym:    symbolize.New(symbolize.SymbolizeOptions{Native: oCfg.Native}),
	}

	return xprocessorhelper.NewProfiles(
		ctx,
		set,
		cfg,
		nextConsumer,
		proc.processProfiles,
		xprocessorhelper.WithCapabilities(proc.capabilities()),
	)
}
