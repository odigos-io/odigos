package odigostailsamplingprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"
)

// Config holds the configuration for the odigostailsampling processor.
type Config struct {

	// the name of the extension that provides the odigos configuration
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`

	// if true, the processor will not sample the traces, but will log the sampling decisions,
	// and forward all traces down the pipeline.
	// this is helpful to evaluate the sampling decisions without actually committing
	// to changes that might drop something important.
	// it also allows to easily view "what would have been dropped" quite easily to troubleshoot issues.
	DryRun bool `mapstructure:"dry_run"`
}

var _ component.Config = (*Config)(nil)

// Validate validates the processor configuration.
func (cfg *Config) Validate() error {
	if cfg.OdigosConfigExtension == nil {
		return errors.New("odigos config extension is required")
	}
	return nil
}
