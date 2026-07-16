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

	"github.com/odigos-io/odigos/common"
	actionsapi "github.com/odigos-io/odigos/common/api/actions"
)

var piiMaskingSupportedSignals = map[common.ObservabilitySignal]struct{}{
	common.TracesObservabilitySignal: {},
}

type PiiMaskingConfig struct {
	AllowAllKeys  bool     `json:"allow_all_keys"`
	BlockedValues []string `json:"blocked_values"`
}

func piiMaskingConfig(cfg []actionsapi.PiiCategory) (PiiMaskingConfig, error) {
	PiiCategories := cfg
	if len(PiiCategories) == 0 {
		return PiiMaskingConfig{}, fmt.Errorf("no PII categories are configured, so this processor is not needed")
	}

	// Allow all attributes to be traced. If set to false it removes all attributes not in allowed_keys which is all attributes
	config := PiiMaskingConfig{
		AllowAllKeys: true,
	}

	for _, piiCategory := range PiiCategories {
		switch piiCategory {
		case actionsapi.CreditCardMasking:
			config.BlockedValues = append(config.BlockedValues, []string{
				"4[0-9]{12}(?:[0-9]{3})?", // Visa credit card number
				"(5[1-5][0-9]{14})",       // MasterCard number
			}...)
		}
	}

	return config, nil
}
