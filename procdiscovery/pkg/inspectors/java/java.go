package java

import (
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"
const JavaVersionRegex = `\d+\.\d+\.\d+\+\d+`

var re = regexp.MustCompile(JavaVersionRegex)

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(proc.ExePath, processName) || strings.Contains(proc.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		javaVersion := re.FindString(value)
		return common.GetVersion(javaVersion)
	}

	return nil
}
