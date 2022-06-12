package inspectors

import (
	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/langDetector/process"
	"strings"
)

type nodejsInspector struct{}

var nodeJs = &nodejsInspector{}

const nodeProcessName = "node"

func (n *nodejsInspector) Inspect(process *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, nodeProcessName) || strings.Contains(process.CmdLine, nodeProcessName) {
		return common.JavascriptProgrammingLanguage, true
	}

	return "", false
}
