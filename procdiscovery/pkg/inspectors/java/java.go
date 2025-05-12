package java

import (
	"bufio"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

var versionRegex = regexp.MustCompile(JavaVersionRegex)

var processNames = []string{
	"java",
}

func (j *JavaInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	if utils.IsBaseExeContainsProcessName(baseExe, processNames) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	mapsFile, err := pcx.GetMapsFile()
	if err != nil {
		return "", false
	}
	scanner := bufio.NewScanner(mapsFile)
	for scanner.Scan() {
		// Check if the shared library "libjvm.so" is loaded in the process memory
		// Ensures "libjvm.so" appears within a path (because of the "/" prefix)
		if strings.Contains(scanner.Text(), "/libjvm.so") {
			return common.JavaProgrammingLanguage, true
		}
	}
	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := pcx.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := versionRegex.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
