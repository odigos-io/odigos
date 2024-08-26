package java

import (
	"github.com/hashicorp/go-version"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"
const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(proc.ExeName, processName) || strings.Contains(proc.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		re := regexp.MustCompile(JavaVersionRegex)
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
