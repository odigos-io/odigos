package golang

import (
	"debug/buildinfo"
	"fmt"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type GolangInspector struct{}

func (g *GolangInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	_, err := buildinfo.ReadFile(file)
	if err != nil {
		return "", false
	}

	return common.GoProgrammingLanguage, true
}
