package php

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PhpInspector struct{}

const processName = "php"

func (n *PhpInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := pcx.Details

	baseExe := filepath.Base(proc.ExePath)

	// Check if baseExe starts with "php"
	if len(baseExe) >= 3 && baseExe[:3] == processName {
		// If it's exactly "php", return true
		if len(baseExe) == 3 {
			return common.PhpProgrammingLanguage, true
		}

		// Use the helper function to check remaining characters
		if utils.IsDigitsOnly(baseExe[3:]) {
			return common.PhpProgrammingLanguage, true
		}
	}

	return "", false
}

func (n *PhpInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
