package golang

import (
	"debug/buildinfo"
	"fmt"
	"github.com/hashicorp/go-version"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
	"regexp"
)

type GolangInspector struct {
	*buildinfo.BuildInfo
}

const GolangVersionRegex = `go(\d+\.\d+\.\d+)`

var re = regexp.MustCompile(GolangVersionRegex)

func (g *GolangInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	file := fmt.Sprintf("/proc/%d/exe", p.ProcessID)
	buildInfo, err := buildinfo.ReadFile(file)
	if err != nil {
		return "", false
	}

	g.BuildInfo = buildInfo

	return common.GoProgrammingLanguage, true
}

func (g *GolangInspector) GetRuntimeVersion(p *process.Details, containerURL string) *version.Version {
	if g.BuildInfo == nil {
		return nil
	}
	match := re.FindStringSubmatch(g.BuildInfo.GoVersion)

	return common.GetVersion(match[1])

}
