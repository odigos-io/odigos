package dotnet

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type DotnetInspector struct{}

var (
	processName = "dotnet"
	binaries    = []string{
		"libcoreclr.so",
	}
)

func (d *DotnetInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	// Only allow exact match for "dotnet"
	if baseExe == processName {
		return common.DotNetProgrammingLanguage, true
	}

	return "", false
}

func (d *DotnetInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	mapsFile, err := pcx.GetMapsFile()
	if err != nil {
		return "", false
	}

	if utils.IsMapsFileContainsBinary(mapsFile, binaries) {
		return common.DotNetProgrammingLanguage, true
	}

	return "", false
}
