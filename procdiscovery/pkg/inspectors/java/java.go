package java

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type JavaInspector struct{}

const processName = "java"

func (j *JavaInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(proc.ExeName, processName) || strings.Contains(proc.CmdLine, processName) {
		return common.JavaProgrammingLanguage, true
	}

	return "", false
}

func (j *JavaInspector) GetRuntimeVersion(proc *process.Details, podIp string) string {
	if value, exists := proc.GetDetailedEnvsValue(process.JavaVersionConst); exists {
		return value
	}

	return ""
}
