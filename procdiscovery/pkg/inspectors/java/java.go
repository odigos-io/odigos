package java

import (
	"regexp"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

// Matches any file path ending with:
//   - "java" (e.g., /usr/bin/java)
//   - "javaw" (though less common on Linux)
//   - "java" / "javaw" followed by version digits (e.g., java8, java11, java17).
var exeRegex = regexp.MustCompile(`.*/javaw?(?:\d+)?$`)
var versionRegex = regexp.MustCompile(JavaVersionRegex)

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if exeRegex.MatchString(proc.ExePath) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := versionRegex.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
