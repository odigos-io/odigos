package golang

import (
	"debug/buildinfo"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type GolangInspector struct{}

func (g *GolangInspector) Inspect(p *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	buildInfo, err := buildinfo.ReadFile(file)
	if err != nil {
		return programLanguageDetails, false
	}

	programLanguageDetails.Language = common.GoProgrammingLanguage
	if buildInfo != nil {
		programLanguageDetails.RuntimeVersion = buildInfo.GoVersion
	}

	return programLanguageDetails, true
}
