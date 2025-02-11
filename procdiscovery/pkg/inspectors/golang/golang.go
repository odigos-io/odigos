package golang

import (
	"debug/buildinfo"
	"regexp"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type GolangInspector struct{}

const GolangVersionRegex = `go(\d+\.\d+\.\d+)`

var re = regexp.MustCompile(GolangVersionRegex)

func (g *GolangInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if p.Exefile == nil {
		return "", false
	}

	_, err := buildinfo.Read(p.Exefile)
	if err != nil {
		return "", false
	}

	return common.GoProgrammingLanguage, true
}

func (g *GolangInspector) GetRuntimeVersion(p *process.Details, containerURL string) *version.Version {
	buildInfo, err := buildinfo.Read(p.Exefile)
	if err != nil || buildInfo == nil {
		return nil
	}
	match := re.FindStringSubmatch(buildInfo.GoVersion)

	return common.GetVersion(match[1])
}
