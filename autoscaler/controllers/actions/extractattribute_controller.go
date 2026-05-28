/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package actions

import (
	"fmt"
	"regexp"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
)

type extractAttributeProcessorConfig struct {
	Extractions []extractAttributeRule `json:"extractions"`
}

type extractAttributeRule struct {
	TargetAttributeName string `json:"target_attribute_name"`
	LookupKey           string `json:"lookup_key,omitempty"`
	DataFormat          string `json:"data_format,omitempty"`
	Regex               string `json:"regex,omitempty"`
}

// extractAttributeConfig translates the API-level ExtractAttributeConfig into the processor-level config and validates cross-field invariants
// that the CRD schema cannot express on its own (LookupKey+DataFormat vs. Regex).
func extractAttributeConfig(cfg *actions.ExtractAttributeConfig) (extractAttributeProcessorConfig, error) {
	if cfg == nil {
		return extractAttributeProcessorConfig{}, fmt.Errorf("extractAttribute config is nil")
	}
	if len(cfg.Extractions) == 0 {
		return extractAttributeProcessorConfig{}, fmt.Errorf("extractions must not be empty")
	}

	config := extractAttributeProcessorConfig{
		Extractions: make([]extractAttributeRule, 0, len(cfg.Extractions)),
	}
	seenNames := make(map[string]int, len(cfg.Extractions))
	for i, extraction := range cfg.Extractions {
		if extraction.TargetAttributeName == "" {
			return config, fmt.Errorf("extractions[%d]: targetAttributeName is required", i)
		}
		if prev, dup := seenNames[extraction.TargetAttributeName]; dup {
			return config, fmt.Errorf("extractions[%d]: duplicate targetAttributeName %q (also used by extractions[%d])", i, extraction.TargetAttributeName, prev)
		}
		seenNames[extraction.TargetAttributeName] = i

		// Check the LookupKey+DataFormat vs. Regex logic - exactly one of them need to be used
		hasLookupKey := extraction.LookupKey != ""
		hasRegex := extraction.Regex != ""

		if hasLookupKey == hasRegex {
			return config, fmt.Errorf("extractions[%d]: exactly one of lookupKey or regex must be set", i)
		}
		if hasLookupKey && extraction.DataFormat == "" {
			return config, fmt.Errorf("extractions[%d]: dataFormat is required when lookupKey is set", i)
		}
		if hasRegex && extraction.DataFormat != "" {
			return config, fmt.Errorf("extractions[%d]: dataFormat must be empty when regex is set", i)
		}
		if hasRegex && extraction.LookupKey != "" {
			return config, fmt.Errorf("extractions[%d]: lookupKey must be empty when regex is set", i)
		}
		if hasRegex {
			if _, err := regexp.Compile(extraction.Regex); err != nil {
				return config, fmt.Errorf("extractions[%d]: invalid regex: %w", i, err)
			}
		}

		config.Extractions = append(config.Extractions, extractAttributeRule{
			TargetAttributeName: extraction.TargetAttributeName,
			LookupKey:           extraction.LookupKey,
			DataFormat:          string(extraction.DataFormat),
			Regex:               extraction.Regex,
		})
	}
	return config, nil
}
