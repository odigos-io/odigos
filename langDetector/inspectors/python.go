package inspectors

import (
	v1 "github.com/keyval-dev/odigos/langDetector/kube/apis/v1"
	"github.com/keyval-dev/odigos/langDetector/process"
	"strings"
)

type pythonInspector struct{}

var python = &pythonInspector{}

const pythonProcessName = "python"

func (p *pythonInspector) Inspect(process *process.Details) (v1.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, pythonProcessName) || strings.Contains(process.CmdLine, pythonProcessName) {
		return v1.PythonProgrammingLanguage, true
	}

	return "", false
}
