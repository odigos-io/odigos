package java

import (
	"bufio"
	"path/filepath"
	"regexp"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

// libjvmRegex is a regular expression that matches any path containing "libjvm.so",
// ensuring that we correctly detect the presence of the JVM shared library.
var libjvmRegex = regexp.MustCompile(`.*/libjvm\.so`)

const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

var versionRegex = regexp.MustCompile(JavaVersionRegex)

// Matches any file path ending with:
//   - "java" (e.g., /usr/bin/java)
//   - "javaw" (though less common on Linux)
//   - "java" / "javaw" followed by version digits (e.g., java8, java11, java17).
var exeRegex = regexp.MustCompile(`^javaw?(?:\d+)?$`)

func (j *JavaInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if exeRegex.MatchString(filepath.Base(pcx.ExePath)) {
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
		if libjvmRegex.MatchString(scanner.Text()) {
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
