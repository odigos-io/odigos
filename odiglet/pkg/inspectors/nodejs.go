package inspectors

import (
	"strings"

	"github.com/keyval-dev/odigos/common"
	procdiscovery "github.com/keyval-dev/odigos/procdiscovery/pkg/process"
)

type nodejsInspector struct{}

var nodeJs = &nodejsInspector{}

const nodeProcessName = "node"

func (n *nodejsInspector) Inspect(process *procdiscovery.Details) (common.ProgrammingLanguage, bool) {
	if strings.Contains(process.ExeName, nodeProcessName) || strings.Contains(process.CmdLine, nodeProcessName) {
		return common.JavascriptProgrammingLanguage, true
	}

	return "", false
}
