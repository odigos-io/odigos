package postgres

import (
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PostgresInspector struct{}

var (
	processNames = []string{"postgres", "postgresql", "psql"}
)

func (n *PostgresInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if utils.IsProcessEqualProcessNames(pcx, processNames) {
		return common.PostgresProgrammingLanguage, true
	}

	return "", false
}

func (n *PostgresInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
