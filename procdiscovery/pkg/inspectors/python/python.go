package python

import (
	"github.com/odigos-io/odigos/common/envs"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PythonInspector struct{}

const pythonProcessName = "python"

func (p *PythonInspector) Inspect(process *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails
	if strings.Contains(process.ExeName, pythonProcessName) || strings.Contains(process.CmdLine, pythonProcessName) {
		programLanguageDetails.Language = common.PythonProgrammingLanguage
		if value, exists := process.Environments.DetailedEnvs[envs.PythonVersionConst]; exists {
			programLanguageDetails.Version = value
		}

		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
