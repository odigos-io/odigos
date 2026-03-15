package odigostailsamplingprocessor

import (
	"errors"

	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/api/sampling"
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

	// Controls whether spans are enhanced with sampling attributes (e.g. category and decisions).
	// These attributes add context when viewing traces and inspecting costs, so you can understand
	// how sampling decisions were made for an individual span and apply changes to fine-tune rules.
	// When dry run is enabled, each span includes the sampling decision (kept or dropped) as it would apply once dry run is disabled.
	SpanSamplingAttributes *sampling.SpanSamplingAttributesConfiguration `mapstructure:"span_sampling_attributes"`

	// Configuration for tail sampling.
	TailSampling *sampling.TailSamplingConfiguration `mapstructure:"tail_sampling"`
}

var _ component.Config = (*Config)(nil)

// Validate validates the processor configuration.
func (cfg *Config) Validate() error {

	if cfg.OdigosConfigExtension == nil {
		return errors.New("odigos config extension is required")
	}

	return nil
}
