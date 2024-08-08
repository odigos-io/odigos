package dotnet

import (
	"fmt"
	"os"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type DotnetInspector struct{}

const (
	aspnet = "ASPNET"
	dotnet = "DOTNET"
)

func (d *DotnetInspector) Inspect(p *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", p.ProcessID))
	if err == nil {
		environ := string(data)
		if strings.Contains(environ, aspnet) || strings.Contains(environ, dotnet) {
			programLanguageDetails.Language = common.DotNetProgrammingLanguage
			return programLanguageDetails, true
		}
	}

	return programLanguageDetails, false
}
