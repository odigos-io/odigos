package nodejs

import (
	"github.com/odigos-io/odigos/common/envs"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type NodejsInspector struct{}

const nodeProcessName = "node"

func (n *NodejsInspector) Inspect(process *process.Details) (common.ProgramLanguageDetails, bool) {
	var programLanguageDetails common.ProgramLanguageDetails

	if strings.Contains(process.ExeName, nodeProcessName) || strings.Contains(process.CmdLine, nodeProcessName) {
		programLanguageDetails.Language = common.JavascriptProgrammingLanguage
		if value, exists := process.Environments.DetailedEnvs[envs.NodeVersionConst]; exists {
			programLanguageDetails.Version = value
		}

		return programLanguageDetails, true
	}

	return programLanguageDetails, false
}
