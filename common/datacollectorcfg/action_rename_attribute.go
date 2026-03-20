package datacollectorcfg

import (
	"fmt"

	"github.com/odigos-io/odigos/common"
)

// RenameAttributeConfig generates a TransformProcessorConfig that renames
// attributes using OTTL set + delete_key statements.
func RenameAttributeConfig(renames map[string]string, signals []common.ObservabilitySignal) (TransformProcessorConfig, error) {
	if len(signals) == 0 {
		return TransformProcessorConfig{}, fmt.Errorf("signals must be set")
	}

	// Every rename produces 2 OTTL statements
	ottlStatements := make([]string, 0, 2*len(renames))
	for from, to := range renames {
		ottlStatements = append(ottlStatements,
			fmt.Sprintf("set(attributes[\"%s\"], attributes[\"%s\"])", to, from),
			fmt.Sprintf("delete_key(attributes, \"%s\")", from),
		)
	}

	cfg := TransformProcessorConfig{
		ErrorMode: "ignore",
	}

	for _, signal := range signals {
		statements := []OttlStatementConfig{
			{Context: "resource", Statements: ottlStatements},
			{Context: "scope", Statements: ottlStatements},
		}

		switch signal {
		case common.LogsObservabilitySignal:
			statements = append(statements, OttlStatementConfig{Context: "log", Statements: ottlStatements})
			cfg.LogStatements = statements
		case common.MetricsObservabilitySignal:
			statements = append(statements, OttlStatementConfig{Context: "datapoint", Statements: ottlStatements})
			cfg.MetricStatements = statements
		case common.TracesObservabilitySignal:
			statements = append(statements,
				OttlStatementConfig{Context: "span", Statements: ottlStatements},
				OttlStatementConfig{Context: "spanevent", Statements: ottlStatements},
			)
			cfg.TraceStatements = statements
		}
	}

	return cfg, nil
}
