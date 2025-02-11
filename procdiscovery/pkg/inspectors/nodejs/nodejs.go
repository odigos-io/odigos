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

func (n *NodejsInspector) Inspect(proc *process.Details) (common.ProgrammingLanguage, bool) {
	if v8Regex.MatchString(filepath.Base(proc.ExePath)) || nodeExecutables[filepath.Base(proc.ExePath)] {
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
