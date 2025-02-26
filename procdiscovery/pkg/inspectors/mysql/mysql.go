package mysql

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type MySQLInspector struct{}

const MySQLProcessName = "mysqld"

func (j *MySQLInspector) QuickScan(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	p := ctx.Details
	if strings.HasSuffix(p.ExePath, MySQLProcessName) || strings.HasSuffix(p.CmdLine, MySQLProcessName) {
		return common.MySQLProgrammingLanguage, true
	}
	return "", false
}

func (j *MySQLInspector) DeepScan(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
