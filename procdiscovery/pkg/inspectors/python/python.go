package python

import (
	"github.com/hashicorp/go-version"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PythonInspector struct{}

const pythonProcessName = "python"

func (p *PythonInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(proc.ExeName, pythonProcessName) || strings.Contains(proc.CmdLine, pythonProcessName) {
		return common.PythonProgrammingLanguage, true
	}

	return "", false
}

func (p *PythonInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.PythonVersionConst); exists {
		return common.GetVersion(value)
	}

	return nil
}
