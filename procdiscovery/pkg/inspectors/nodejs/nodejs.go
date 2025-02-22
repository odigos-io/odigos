package nodejs

import (
	"path/filepath"
	"regexp"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

var v8Regex = regexp.MustCompile(`^(?:.*/)?node(\d+)?$`)

var nodeExecutables = map[string]bool{
	"npm":  true,
	"yarn": true,
}

// LightCheck uses the filename heuristics to detect a Node.js process.
func (n *NodejsInspector) LightCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := ctx.Details
	baseExe := filepath.Base(proc.ExePath)
	if v8Regex.MatchString(baseExe) || nodeExecutables[baseExe] {
		return common.JavascriptProgrammingLanguage, true
	}
	return "", false
}

// ExpensiveCheck is not required for Node.js detection, so it returns no detection.
func (n *NodejsInspector) ExpensiveCheck(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *NodejsInspector) GetRuntimeVersion(ctx *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := ctx.Details.GetDetailedEnvsValue(process.NodeVersionConst); exists {
		return common.GetVersion(value)
	}

	return nil
}
