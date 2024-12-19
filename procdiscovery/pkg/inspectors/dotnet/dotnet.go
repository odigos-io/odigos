package dotnet

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type DotnetInspector struct{}

func (d *DotnetInspector) Inspect(p *process.Details) (common.ProgrammingLanguage, bool) {
	mapsPath := filepath.Join("/proc", strconv.Itoa(p.ProcessID), "maps")
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
