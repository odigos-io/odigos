package dotnet

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type DotnetInspector struct{}

const processName = "dotnet"

const processName = "dotnet"

func (d *DotnetInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if strings.HasPrefix(filepath.Base(pcx.ExePath), processName) {
		return common.DotNetProgrammingLanguage, true
	}
	return "", false
}
func (d *DotnetInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// Heavy check: read the process maps from cache and look for "libcoreclr.so"
	mapsFile, err := pcx.GetMapsFile()
	if err != nil {
		return "", false
	}

	// Scan the maps content for the .NET runtime library.
	scanner := bufio.NewScanner(mapsFile)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "libcoreclr.so") {
			return common.DotNetProgrammingLanguage, true
		}
	}

	return "", false
}
