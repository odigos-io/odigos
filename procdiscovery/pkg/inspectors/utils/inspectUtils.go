package utils

import (
	"bufio"
	"strings"

	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

func IsBaseExeContainsProcessName(baseExe string, processNames []string) bool {
	for _, processName := range processNames {
		baseLen := len(baseExe)
		procLen := len(processName)

		// Check if baseExe starts with processName
		if baseLen >= procLen && baseExe[:procLen] == processName {
			// If it's exactly processName, or only digits follow
			if baseLen == procLen || IsDigitsOnly(baseExe[procLen:]) {
				return true
			}
		}
	}

	return false
}

func IsDigitsOnly(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
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
