package inspectors

import (
	"strings"

	"github.com/keyval-dev/odigos/common"
	procdiscovery "github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type pythonInspector struct{}

var python = &pythonInspector{}

const pythonProcessName = "python"

func (p *pythonInspector) Inspect(process *procdiscovery.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, pythonProcessName) || strings.Contains(process.CmdLine, pythonProcessName) {
		return common.PythonProgrammingLanguage, true
	}

	return "", false
}
