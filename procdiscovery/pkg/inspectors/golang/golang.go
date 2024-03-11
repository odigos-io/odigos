package golang

import (
	"debug/buildinfo"
	"fmt"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/process"
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
