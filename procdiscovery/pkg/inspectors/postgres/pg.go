package postgres

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type PostgresInspector struct{}

const PostgreSQLProcessName = "postgres"

func (j *PostgresInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.HasSuffix(p.ExeName, PostgreSQLProcessName) || strings.HasSuffix(p.CmdLine, PostgreSQLProcessName) {
		return common.PostgresProgrammingLanguage, true
	}

	return "", false
}
