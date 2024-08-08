package java

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"

func (j *JavaInspector) Inspect(p *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails

	if strings.Contains(p.ExeName, processName) || strings.Contains(p.CmdLine, processName) {
		programLanguageDetails.Language = common.JavaProgrammingLanguage
		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
