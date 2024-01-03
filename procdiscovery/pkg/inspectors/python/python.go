package python

import (
	"strings"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type PythonInspector struct{}

const pythonProcessName = "python"

func (p *PythonInspector) Inspect(process *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, pythonProcessName) || strings.Contains(process.CmdLine, pythonProcessName) {
		return common.PythonProgrammingLanguage, true
	}

	return "", false
}
