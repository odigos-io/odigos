package mysql

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type MySQLInspector struct{}

const MySQLProcessName = "mysqld"

func (j *MySQLInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	p := pcx.Details
	if filepath.Base(p.ExePath) == MySQLProcessName {
		return common.MySQLProgrammingLanguage, true
	}
	return "", false
}

func (j *MySQLInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
