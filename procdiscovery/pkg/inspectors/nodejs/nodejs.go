package nodejs

import (
	"github.com/hashicorp/go-version"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

const nodeProcessName = "node"

func (n *NodejsInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(proc.ExeName, nodeProcessName) || strings.Contains(proc.CmdLine, nodeProcessName) {
		return common.JavascriptProgrammingLanguage, true
	}

	return "", false
}

func (n *NodejsInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.NodeVersionConst); exists {
		return common.GetVersion(value)
	}

	return nil
}
