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

	// Check if baseExe starts with "node"
	if len(baseExe) >= 4 && baseExe[:4] == "node" {
		// If it's exactly "node", return true
		if len(baseExe) == 4 {
			return common.JavascriptProgrammingLanguage, true
		}

		// Use the helper function to check remaining characters
		if utils.IsDigitsOnly(baseExe[4:]) {
			return common.JavascriptProgrammingLanguage, true
		}
		return "", false
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
