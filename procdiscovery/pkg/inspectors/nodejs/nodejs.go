package nodejs

import (
	"strings"

	"github.com/keyval-dev/odigos/common"
	"github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

const nodeProcessName = "node"

func (n *NodejsInspector) Inspect(process *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, nodeProcessName) || strings.Contains(process.CmdLine, nodeProcessName) {
		return common.JavascriptProgrammingLanguage, true
	}

	return "", false
}
