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

// LightCheck performs a lightweight check by inspecting the process's executable path and command line.
func (j *MySQLInspector) LightCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if filepath.Base(ctx.ExePath) == "mysqld" {
		return common.MySQLProgrammingLanguage, true
	}
	return "", false
}

// ExpensiveCheck is not needed for MySQL detection so it returns no detection.
func (j *MySQLInspector) ExpensiveCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
