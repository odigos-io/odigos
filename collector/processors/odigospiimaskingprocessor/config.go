package odigospiimaskingprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/confmap/xconfmap"
)

type Config struct {
	// OdigosConfigExtension provides per-workload PII masking options from the
	// extension cache (e.g. odigos_config_k8s). Must implement OdigosConfigExtension.
	OdigosConfigExtension *component.ID `mapstructure:"odigos_config_extension"`
}

var _ xconfmap.Validator = (*Config)(nil)

func (cfg Config) Validate() error {
	if cfg.OdigosConfigExtension == nil {
		return fmt.Errorf("odigos_config_extension is required")
	}
	typeStr := cfg.OdigosConfigExtension.Type().String()
	if _, err := component.NewType(typeStr); err != nil {
		return fmt.Errorf("invalid odigos_config_extension type %q: %w", typeStr, err)
	}
	return nil
}
