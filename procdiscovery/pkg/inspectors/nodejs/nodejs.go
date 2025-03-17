package nodejs

import (
	"path/filepath"
	"strconv"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

var nodeExecutables = map[string]bool{
	"npm":  true,
	"yarn": true,
}

func (n *NodejsInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := pcx.Details
	baseExe := filepath.Base(proc.ExePath)

	// Check if the string starts with "node"
	if strings.HasPrefix(baseExe, "node") {
		// If the string is exactly "node", return true
		if len(baseExe) == 4 {
			return common.JavascriptProgrammingLanguage, true
		}

		// If there's extra text after "node", verify it's purely numeric (e.g., "node10", "node16")
		if _, err := strconv.Atoi(baseExe[4:]); err == nil {
			return common.JavascriptProgrammingLanguage, true
		}

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
