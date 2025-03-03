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

func (n *NodejsInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := pcx.Details
	baseExe := filepath.Base(proc.ExePath)
	if v8Regex.MatchString(baseExe) || nodeExecutables[baseExe] {
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
