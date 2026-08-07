package odigosextractattributeprocessor

import (
	"fmt"

	"go.opentelemetry.io/collector/component"
)

type DataFormat string

const (
	FormatUnset        DataFormat = ""
	FormatResourcePath DataFormat = "resource_path"
	FormatJSON         DataFormat = "json"
	FormatSQL          DataFormat = "sql"
)

// Extraction is one self-contained extraction rule.
// It uses either a preset pattern (LookupKey+DataFormat) or a custom Regex, and writes the captured value
// to a new span attribute named TargetAttributeName.
type Extraction struct {
	TargetAttributeName string     `mapstructure:"target_attribute_name"`
	LookupKey           string     `mapstructure:"lookup_key"`
	DataFormat          DataFormat `mapstructure:"data_format"`
	Regex               string     `mapstructure:"regex"`
}

type Config struct {
	Extractions []Extraction `mapstructure:"extractions"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	if len(cfg.Extractions) == 0 {
		return fmt.Errorf("extractions must not be empty")
	}

	seenNames := make(map[string]int, len(cfg.Extractions))
	for i, extraction := range cfg.Extractions {
		if extraction.TargetAttributeName == "" {
			return fmt.Errorf("extractions[%d]: target_attribute_name is required", i)
		}
		// Make sure we don't have extractions writing to the same new attribute name
		if prev, dup := seenNames[extraction.TargetAttributeName]; dup {
			return fmt.Errorf("extractions[%d]: duplicate target_attribute_name %q (also used by extractions[%d])", i, extraction.TargetAttributeName, prev)
		}
		seenNames[extraction.TargetAttributeName] = i

		hasLookupKey := extraction.LookupKey != ""
		hasRegex := extraction.Regex != ""

		if hasLookupKey && hasRegex {
			return fmt.Errorf("extractions[%d]: cannot set both lookup_key and regex - choose one", i)
		}
		if !hasLookupKey && !hasRegex {
			return fmt.Errorf("extractions[%d]: must set either lookup_key or regex", i)
		}

		if hasLookupKey {
			switch extraction.DataFormat {
			case FormatResourcePath, FormatJSON, FormatSQL:
			case FormatUnset:
				return fmt.Errorf("extractions[%d]: data_format is required when lookup_key is set", i)
			default:
				return fmt.Errorf("extractions[%d]: invalid data_format %q (must be %q, %q, or %q)",
					i, extraction.DataFormat, FormatResourcePath, FormatJSON, FormatSQL)
			}
		}

		if hasRegex {
			if extraction.DataFormat != FormatUnset {
				return fmt.Errorf("extractions[%d]: data_format must not be set when using regex", i)
			}
			if extraction.LookupKey != "" {
				return fmt.Errorf("extractions[%d]: lookup_key must not be set when using regex", i)
			}
		}
	}

	return nil
}
