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

func (d *DotnetInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/environ", p.ProcessID))
	if err == nil {
		environ := string(data)
		if strings.Contains(environ, aspnet) || strings.Contains(environ, dotnet) {
			return common.DotNetProgrammingLanguage, true
		}
	}

	return "", false
}
