package inspectors

import (
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/langDetector/process"
	"strings"
)

type pythonInspector struct{}

var python = &pythonInspector{}

const pythonProcessName = "python"

func (p *pythonInspector) Inspect(process *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, pythonProcessName) || strings.Contains(process.CmdLine, pythonProcessName) {
		return common.PythonProgrammingLanguage, true
	}

	return "", false
}
