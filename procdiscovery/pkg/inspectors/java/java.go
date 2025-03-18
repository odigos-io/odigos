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

func (j *JavaInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	// Check if baseExe starts with "java"
	if len(baseExe) >= 4 && baseExe[:4] == "java" {
		// If it's exactly "java", return true
		if len(baseExe) == 4 {
			return common.JavaProgrammingLanguage, true
		}

		// Use IsDigitsOnly from utils to ensure all remaining characters are digits
		if utils.IsDigitsOnly(baseExe[4:]) {
			return common.JavaProgrammingLanguage, true
		}
		return "", false
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
	if value, exists := pcx.Details.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := versionRegex.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
