package odigospiimaskingprocessor

import (
	"fmt"
	"regexp"

	"go.opentelemetry.io/collector/component"

	"github.com/odigos-io/odigos/common/api/actions"
)

type Config struct {
	actions.PiiMaskingConfig `mapstructure:",squash"`
}

var _ component.Config = (*Config)(nil)

func (cfg *Config) Validate() error {
	for i, category := range cfg.PiiCategories {
		if _, ok := categoryMasks[category]; !ok {
			return fmt.Errorf("piiCategories[%d]: unsupported category %q", i, category)
		}
	}

	for i, masking := range cfg.CustomFormatMaskings {
		if masking.LookupKey == "" {
			return fmt.Errorf("customFormatMaskings[%d]: lookupKey is required", i)
		}
		switch masking.DataFormat {
		case actions.FormatJSON, actions.FormatSQL, actions.FormatResourcePath:
		case "":
			return fmt.Errorf("customFormatMaskings[%d]: dataFormat is required", i)
		default:
			return fmt.Errorf("customFormatMaskings[%d]: unsupported dataFormat %q", i, masking.DataFormat)
		}
	}

	for i, masking := range cfg.CustomRegexMaskings {
		if masking.Regex == "" {
			return fmt.Errorf("customRegexMaskings[%d]: regex is required", i)
		}
		re, err := regexp.Compile(masking.Regex)
		if err != nil {
			return fmt.Errorf("customRegexMaskings[%d]: invalid regex: %w", i, err)
		}
		if re.NumSubexp() < 1 {
			return fmt.Errorf("customRegexMaskings[%d]: regex must contain at least one capture group", i)
		}
	}

	return nil
}
