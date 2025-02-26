package dotnet

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type DotnetInspector struct{}

const processName = "dotnet"

func (d *DotnetInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	if filepath.Base(p.ExePath) == processName {
		return common.DotNetProgrammingLanguage, true
	}

	mapsPath := fmt.Sprintf("/proc/%d/maps", p.ProcessID)
	f, err := os.Open(mapsPath)
	if err != nil {
		return "", false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// Check if the .NET core runtime library is present
		if strings.Contains(line, "libcoreclr.so") {
			return common.DotNetProgrammingLanguage, true
		}
	}

	return "", false
}
