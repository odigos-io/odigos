package nodejs

import (
	"path/filepath"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

var nodeExecutables = map[string]bool{
	"npm":  true,
	"npx":  true,
	"yarn": true,
}

func (n *NodejsInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := pcx.Details
	baseExe := filepath.Base(proc.ExePath)

	if utils.IsBaseExeContainsProcessName(baseExe, "node") {
		return common.JavascriptProgrammingLanguage, true
	}

	// Check if the executable is a recognized Node.js package manager (npm, yarn)
	if nodeExecutables[baseExe] {
		return common.JavascriptProgrammingLanguage, true
	}

	return "", false
}

func (n *NodejsInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *NodejsInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := pcx.Details.GetDetailedEnvsValue(process.NodeVersionConst); exists {
		return common.GetVersion(value)
	}

	return nil
}
