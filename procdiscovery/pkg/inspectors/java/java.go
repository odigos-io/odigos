package java

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails

	if strings.Contains(proc.ExeName, processName) || strings.Contains(proc.CmdLine, processName) {
		programLanguageDetails.Language = common.JavaProgrammingLanguage
		if proc.Environments != nil {
			if value, exists := proc.Environments.DetailedEnvs[process.JavaVersionConst]; exists {
				programLanguageDetails.RuntimeVersion = value
			}
		}

		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
