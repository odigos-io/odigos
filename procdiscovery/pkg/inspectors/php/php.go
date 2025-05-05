package php

import (
	"fmt"
	"os"
	"regexp"

	"path/filepath"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PhpInspector struct{}

var processNames = []string{
	"php",
	"php-fpm",
}

var versionRegex = regexp.MustCompile(`X-Powered-By:\s*PHP/(\d+\.\d+\.\d+)`)

func (n *PhpInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	if utils.IsBaseExeContainsProcessName(baseExe, processNames) {
		return common.PhpProgrammingLanguage, true
	}

	return "", false
}

func (n *PhpInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *PhpInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := pcx.Details.GetDetailedEnvsValue(process.PhpVersionConst); exists {
		return common.GetVersion(value)
	}

	vers := getVersionFromBinary(pcx.Details.ProcessID)
	if vers != "" {
		return common.GetVersion(vers)
	}

	return nil
}

func getVersionFromBinary(pid int) string {
	paths := []string{
		fmt.Sprintf("/proc/%d/root/usr/local/bin/php", pid),
		fmt.Sprintf("/proc/%d/root/usr/bin/php", pid),
		fmt.Sprintf("/proc/%d/root/usr/local/sbin/php-fpm", pid),
		fmt.Sprintf("/proc/%d/root/usr/sbin/php-fpm", pid),
	}

	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		for _, line := range extractStringFromBinary(data) {
			if matches := versionRegex.FindStringSubmatch(line); matches != nil {
				return matches[1]
			}
		}
	}

	return ""
}

func extractStringFromBinary(data []byte) []string {
	var result []string
	var current []byte

	for _, b := range data {
		if b >= 32 && b <= 126 {
			current = append(current, b)
		} else if len(current) >= 4 {
			result = append(result, string(current))
			current = nil
		} else {
			current = nil
		}
	}
	if len(current) >= 4 {
		result = append(result, string(current))
	}

	return result
}
