package nodejs

import (
	"path/filepath"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

const nodeProcessName = "node"

var nodeExecutables = map[string]bool{
	"node": true,
	"npm":  true,
	"yarn": true,
}

func (n *NodejsInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if nodeExecutables[filepath.Base(proc.ExePath)] {
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
