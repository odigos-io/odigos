package postgres

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PostgresInspector struct{}

var (
	processNames = []string{"postgres", "postgresql", "psql"}
)

func (n *PostgresInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	if utils.IsBaseExeContainsProcessName(baseExe, processNames) {
		return common.PostgresProgrammingLanguage, true
	}

	return "", false
}

func (n *PostgresInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
