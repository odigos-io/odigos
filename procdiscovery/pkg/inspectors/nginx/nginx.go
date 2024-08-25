package nginx

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// This is an experimental feature, It is not a language
// but in order to avoid huge refactoring we are adding it as a language for now
type NginxInspector struct{}

const NginxProcessName = "nginx"

func (j *NginxInspector) Inspect(p *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails
	if strings.Contains(p.CmdLine, NginxProcessName) || strings.Contains(p.ExeName, NginxProcessName) {
		programLanguageDetails.Language = common.NginxProgrammingLanguage
		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
