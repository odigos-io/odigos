package nodejs

import (
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

const nodeProcessName = "node"

func (n *NodejsInspector) Inspect(proc *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails

	if strings.Contains(proc.ExeName, nodeProcessName) || strings.Contains(proc.CmdLine, nodeProcessName) {
		programLanguageDetails.Language = common.JavascriptProgrammingLanguage
		if value, exists := proc.GetDetailedEnvsValue(process.NodeVersionConst); exists {
			programLanguageDetails.RuntimeVersion = value
		}

		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
