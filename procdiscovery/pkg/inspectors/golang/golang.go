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

// LightCheck performs a lightweight check for Go by reading the build info.
func (g *GolangInspector) LightCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {

	ctx.ExeContent()
	if ctx.ExeFileContent == nil {
		return "", false
	}

	// Try reading the build info. If successful, this is a Go binary.
	if _, err := buildinfo.Read(ctx.ExeFileContent); err != nil {
		return "", false
	}

	return common.GoProgrammingLanguage, true
}

// ExpensiveCheck returns false because no heavy check is required for Go.
func (g *GolangInspector) ExpensiveCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (g *GolangInspector) GetRuntimeVersion(ctx *process.ProcessContext, containerURL string) *version.Version {
	buildInfo, err := buildinfo.Read(ctx.ExeFileContent)
	if err != nil || buildInfo == nil {
		return nil
	}
	match := re.FindStringSubmatch(buildInfo.GoVersion)

	return common.GetVersion(match[1])
}
