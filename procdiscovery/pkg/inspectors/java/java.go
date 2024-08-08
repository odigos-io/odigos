package java

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"

func (j *JavaInspector) Inspect(process *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails

	if strings.Contains(process.ExeName, processName) || strings.Contains(process.CmdLine, processName) {
		programLanguageDetails.Language = common.JavaProgrammingLanguage
		if value, exists := process.Environments.DetailedEnvs[process.PythonVersionConst]; exists {
			programLanguageDetails.RuntimeVersion = value
		}
		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
