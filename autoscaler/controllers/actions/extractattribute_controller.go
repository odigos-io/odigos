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

	"github.com/odigos-io/odigos/api/odigos/v1alpha1/actions"
)

type extractAttributeProcessorConfig struct {
	Extractions []extractAttributeRule `json:"extractions"`
}

type extractAttributeRule struct {
	Target     string `json:"target"`
	Source     string `json:"source,omitempty"`
	DataFormat string `json:"data_format,omitempty"`
	Regex      string `json:"regex,omitempty"`
}

// extractAttributeConfig translates the API-level ExtractAttributeConfig into the processor-level config and validates cross-field invariants
// that the CRD schema cannot express on its own (Source+DataForm vs. Regex).
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
	seenTargets := make(map[string]int, len(cfg.Extractions))
	for i, extraction := range cfg.Extractions {
		if extraction.Target == "" {
			return config, fmt.Errorf("extractions[%d]: target is required", i)
		}
		if prev, dup := seenTargets[extraction.Target]; dup {
			return config, fmt.Errorf("extractions[%d]: duplicate target %q (also used by extractions[%d])", i, extraction.Target, prev)
		}
		seenTargets[extraction.Target] = i

		// Check the Source+DataForm vs. Regex logic - exactly one of them need to be used
		hasSource := extraction.Source != ""
		hasRegex := extraction.Regex != ""

		if hasSource == hasRegex {
			return config, fmt.Errorf("extractions[%d]: exactly one of source or regex must be set", i)
		}
		if hasSource && extraction.DataFormat == "" {
			return config, fmt.Errorf("extractions[%d]: dataFormat is required when source is set", i)
		}
		if hasRegex && extraction.DataFormat != "" {
			return config, fmt.Errorf("extractions[%d]: dataFormat must be empty when regex is set", i)
		}
		if hasRegex && extraction.Source != "" {
			return config, fmt.Errorf("extractions[%d]: source must be empty when regex is set", i)
		}

		config.Extractions = append(config.Extractions, extractAttributeRule{
			Target:     extraction.Target,
			Source:     extraction.Source,
			DataFormat: string(extraction.DataFormat),
			Regex:      extraction.Regex,
		})
	}
	return config, nil
}
