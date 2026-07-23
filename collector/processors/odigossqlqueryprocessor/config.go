package odigossqlqueryprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config struct {
	// InferAttributes and RedactLiterals are legacy static options used when
	// odigos_config_extension is unset. Prefer per-source config from the extension.
	InferAttributes bool `mapstructure:"infer_attributes"`
	RedactLiterals  bool `mapstructure:"redact_literals"`

	// OdigosConfigExtension is the default for Odigos: per-workload SQL query options
	// from the extension cache (e.g. odigos_config_k8s). Must implement OdigosConfigExtension.
	// If omitted, InferAttributes / RedactLiterals above are used (legacy).
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

var _ xconfmap.Validator = (*Config)(nil)

func (c Config) Validate() error {
	if c.OdigosConfigExtension != nil {
		typeStr := c.OdigosConfigExtension.Type().String()
		if _, err := component.NewType(typeStr); err != nil {
			return fmt.Errorf("invalid odigos_config_extension type %q: %w", typeStr, err)
		}
	}
	return nil
}
