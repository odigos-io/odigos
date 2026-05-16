package odigosextractattributeprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

type DataFormat string

const (
	FormatUnset DataFormat = ""
	FormatURL   DataFormat = "url"
	FormatJSON  DataFormat = "json"
)

// Extraction is one self-contained extraction rule.
// It uses either a preset pattern (Source+DataFormat) or a custom Regex, and writes the captured value to Target.
type Extraction struct {
	Target     string     `mapstructure:"target"`
	Source     string     `mapstructure:"source"`
	DataFormat DataFormat `mapstructure:"data_format"`
	Regex      string     `mapstructure:"regex"`
}

type Config struct {
	Extractions []Extraction `mapstructure:"extractions"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if len(cfg.Extractions) == 0 {
		return fmt.Errorf("extractions must not be empty")
	}

	seenTargets := make(map[string]int, len(cfg.Extractions))
	for i, extraction := range cfg.Extractions {
		if extraction.Target == "" {
			return fmt.Errorf("extractions[%d]: target is required", i)
		}
		// Make sure we don't have extractions with the same targets
		if prev, dup := seenTargets[extraction.Target]; dup {
			return fmt.Errorf("extractions[%d]: duplicate target %q (also used by extractions[%d])", i, extraction.Target, prev)
		}
		seenTargets[extraction.Target] = i

		hasSource := extraction.Source != ""
		hasRegex := extraction.Regex != ""

		if hasSource && hasRegex {
			return fmt.Errorf("extractions[%d]: cannot set both source and regex - choose one", i)
		}
		if !hasSource && !hasRegex {
			return fmt.Errorf("extractions[%d]: must set either source or regex", i)
		}

		if hasSource {
			switch extraction.DataFormat {
			case FormatURL, FormatJSON:
			case FormatUnset:
				return fmt.Errorf("extractions[%d]: data_format is required when source is set", i)
			default:
				return fmt.Errorf("extractions[%d]: invalid data_format %q (must be %q or %q)",
					i, extraction.DataFormat, FormatURL, FormatJSON)
			}
		}

		if hasRegex {
			if extraction.DataFormat != FormatUnset {
				return fmt.Errorf("extractions[%d]: data_format must not be set when using regex", i)
			}
			if extraction.Source != "" {
				return fmt.Errorf("extractions[%d]: source must not be set when using regex", i)
			}
		}
	}

	return nil
}
