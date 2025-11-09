package utils

import (
	"bufio"
	"path/filepath"
	"slices"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func IsProcessEqualProcessNames(pcx *process.ProcessContext, processNames []string) bool {
	baseExe := filepath.Base(pcx.ExePath)

	return slices.Contains(processNames, baseExe)
}

func IsMapsFileContainsBinary(mapsFile process.ProcessFile, binaries []string) bool {
	scanner := bufio.NewScanner(mapsFile)

	for scanner.Scan() {
		line := scanner.Text()

		for _, binary := range binaries {
			if strings.Contains(line, binary) {
				return true
			}
		}
	}

	return false
}
