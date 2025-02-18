package dotnet

import (
	"bufio"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type DotnetInspector struct{}

func (d *DotnetInspector) InspectLow(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// No low-cost heuristic; immediately defer to the heavy check.
	return "", false
}

func (d *DotnetInspector) InspectHeavy(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	// Heavy check: read the process maps from cache and look for "libcoreclr.so"
	maps := ctx.MapsContent()
	if maps == nil {
		return "", false
	}

	// Scan the maps content for the .NET runtime library.
	scanner := bufio.NewScanner(maps)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "libcoreclr.so") {
			return common.DotNetProgrammingLanguage, true
		}
	}

	return "", false
}

// func (d *DotnetInspector) Inspect(processContext *process.ProcessContext) (common.ProgrammingLanguage, bool) {
// 	// Retrieve the cached maps file content.
// 	maps := processContext.MapsContent()
// 	if maps == nil {
// 		return "", false
// 	}

// 	// Scan the maps content for the .NET runtime library.
// 	scanner := bufio.NewScanner(maps)
// 	for scanner.Scan() {
// 		line := scanner.Text()
// 		if strings.Contains(line, "libcoreclr.so") {
// 			return common.DotNetProgrammingLanguage, true
// 		}
// 	}

// 	return "", false
// }
