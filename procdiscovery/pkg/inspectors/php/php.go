package php

import (
	"path/filepath"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PhpInspector struct{}

func (n *PhpInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := pcx.Details
	baseExe := filepath.Base(proc.ExePath)

	if utils.IsBaseExeMatchProcessName(baseExe, "php") {
		return common.PhpProgrammingLanguage, true
	}

	return "", false
}

func (n *PhpInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}
