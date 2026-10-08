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
)

func deleteAttributeConfig(cfg []string, signals []common.ObservabilitySignal) (TransformProcessorConfig, error) {
	config := TransformProcessorConfig{
		ErrorMode: "propagate", // deleting attributes is a security sensitive operation, so we should propagate errors
	}

	if signals == nil {
		return config, fmt.Errorf("Signals must be set")
	}

	ottlDeleteKeyStatements := make([]string, len(cfg))
	for i, attr := range cfg {
		ottlDeleteKeyStatements[i] = fmt.Sprintf("delete_key(attributes, \"%s\")", attr)
	}

	for _, signal := range signals {
		switch signal {

		case common.LogsObservabilitySignal:
			config.LogStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "log",
					Statements: ottlDeleteKeyStatements,
				},
			}

		case common.MetricsObservabilitySignal:
			config.MetricStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "datapoint",
					Statements: ottlDeleteKeyStatements,
				},
			}

		case common.TracesObservabilitySignal:
			config.TraceStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "span",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "spanevent",
					Statements: ottlDeleteKeyStatements,
				},
			}

		case common.ProfilesObservabilitySignal:
			// resource, scope and profile are the only OTTL contexts the transformprocessor accepts for profiles.
			config.ProfileStatements = []OttlStatementConfig{
				{
					Context:    "resource",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "scope",
					Statements: ottlDeleteKeyStatements,
				},
				{
					Context:    "profile",
					Statements: ottlDeleteKeyStatements,
				},
			}
		}
	}
	return config, nil
}
