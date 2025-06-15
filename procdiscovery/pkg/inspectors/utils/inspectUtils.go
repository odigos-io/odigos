package utils

import (
	"path/filepath"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func IsProcessEqualProcessNames(pcx *process.ProcessContext, processNames []string) bool {
	baseExe := filepath.Base(pcx.ExePath)

	for _, processName := range processNames {
		if baseExe == processName {
			return true
		}
	}

	return false
}
