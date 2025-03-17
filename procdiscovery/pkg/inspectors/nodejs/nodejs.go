package nodejs

import (
	"path/filepath"
	"strings"
	"unicode"

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

	// Check if the executable is:
	// - "node" (exact match)
	// - "node" followed by digits (e.g., "node8", "node10", etc.)
	// - One of the recognized package managers: "npm" or "yarn"
	//
	// The check:
	// - `strings.HasPrefix(baseExe, "node")` ensures it starts with "node".
	// - `len(baseExe) == 4` allows "node" as a standalone executable.
	// - `unicode.IsDigit(rune(baseExe[4]))` ensures that if there’s an extra character (char at the 5th position), it's a number (rejecting cases like "nodejs").
	if strings.HasPrefix(baseExe, "node") &&
		(len(baseExe) == 4 || unicode.IsDigit(rune(baseExe[4]))) ||
		nodeExecutables[baseExe] {
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
