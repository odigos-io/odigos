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
	"slices"

	"github.com/odigos-io/odigos/common"
)

func renameAttributeConfig(cfg map[string]string, signals []common.ObservabilitySignal) (TransformProcessorConfig, error) {
	config := TransformProcessorConfig{
		ErrorMode: "ignore",
	}

	if signals == nil {
		return TransformProcessorConfig{}, fmt.Errorf("Signals must be set")
	}

	// Sort keys so Processor Spec is stable across reconciles. Ranging a map is
	// non-deterministic and would keep SSA-patching the owned Processor forever.
	keys := make([]string, 0, len(cfg))
	for from := range cfg {
		keys = append(keys, from)
	}
	slices.Sort(keys)

	ottlStatements := make([]string, 0, 2*len(cfg))
	for _, from := range keys {
		to := cfg[from]
		ottlStatements = append(ottlStatements,
			fmt.Sprintf("set(attributes[\"%s\"], attributes[\"%s\"])", to, from),
			fmt.Sprintf("delete_key(attributes, \"%s\")", from),
		)
	}

	for _, signal := range signals {
		switch signal {

		case common.LogsObservabilitySignal:
			config.LogStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "log",
					Statements: ottlStatements,
				},
			}

		case common.MetricsObservabilitySignal:
			config.MetricStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "datapoint",
					Statements: ottlStatements,
				},
			}

		case common.TracesObservabilitySignal:
			config.TraceStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "span",
					Statements: ottlStatements,
				},
				{
					Context:    "spanevent",
					Statements: ottlStatements,
				},
			}

		case common.ProfilesObservabilitySignal:
			// resource, scope and profile are the only OTTL contexts the transformprocessor accepts for profiles.
			config.ProfileStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlStatements,
				},
				{
					Context:    "scope",
					Statements: ottlStatements,
				},
				{
					Context:    "profile",
					Statements: ottlStatements,
				},
			}
		}
	}
	return config, nil
}
