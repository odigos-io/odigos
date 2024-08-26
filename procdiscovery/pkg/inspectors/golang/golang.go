package golang

import (
	"debug/buildinfo"
	"fmt"
	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"regexp"
)

type GolangInspector struct{}

const GolangVersionRegex = `go(\d+\.\d+\.\d+)`

func (g *GolangInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	_, err := buildinfo.ReadFile(file)
	if err != nil {
		return "", false
	}

	return common.GoProgrammingLanguage, true
}

func (g *GolangInspector) GetRuntimeVersion(p *process.Details, containerURL string) *version.Version {
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	buildInfo, err := buildinfo.ReadFile(file)
	if err != nil || buildInfo == nil {
		return nil
	}

	re := regexp.MustCompile(GolangVersionRegex)
	match := re.FindStringSubmatch(buildInfo.GoVersion)

	return common.GetVersion(match[1])

}
