package python

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PythonInspector struct{}

const pythonProcessName = "python"

func (p *PythonInspector) Inspect(proc *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails
	if strings.Contains(proc.ExeName, pythonProcessName) || strings.Contains(proc.CmdLine, pythonProcessName) {
		programLanguageDetails.Language = common.PythonProgrammingLanguage
		if value, exists := proc.GetDetailedEnvsValue(process.PythonVersionConst); exists {
			programLanguageDetails.RuntimeVersion = value
		}

		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
