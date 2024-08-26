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

func (j *MySQLInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.HasSuffix(p.ExeName, MySQLProcessName) || strings.HasSuffix(p.CmdLine, MySQLProcessName) {
		return common.MySQLProgrammingLanguage, true
	}

	return "", false
}
