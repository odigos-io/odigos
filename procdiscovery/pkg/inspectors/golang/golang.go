package golang

import (
	"debug/buildinfo"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type GolangInspector struct{}

var (
	falseProcessNames = []string{"thrust"}
	versionRegex      = regexp.MustCompile(`go(\d+\.\d+\.\d+)`)
)

func (g *GolangInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// 1st:
	// Check if the process name is in the false list.
	// This is to avoid false positives for Go processes that init or wrap other processes that are not actually Go applications.
	if utils.IsProcessEqualProcessNames(pcx, falseProcessNames) {
		return "", false
	}

	// 2nd:
	// Check if the process is actually a Go binary (not in the false list).
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return "", false
	}

	// Try reading the build info. If successful, this is a Go binary.
	if _, err := buildinfo.Read(exeFile); err != nil {
		// DynatraceDynamizerExeSubString is wrapper exe for dynatrace agent for go only
		if pcx != nil && strings.Contains(pcx.ExePath, process.DynatraceDynamizerExeSubString) {
			return common.GoProgrammingLanguage, true
		}
		return "", false
	}

	return common.GoProgrammingLanguage, true
}

func (g *GolangInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (g *GolangInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return nil
	}
	buildInfo, err := buildinfo.Read(exeFile)
	if err != nil || buildInfo == nil {
		return nil
	}
	match := versionRegex.FindStringSubmatch(buildInfo.GoVersion)

	return common.GetVersion(match[1])
}
