package php

import (
	"debug/elf"
	"io"
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

	vers := getVersionFromExecutable(pcx)
	if vers != "" {
		return common.GetVersion(vers)
	}

	return nil
}

func getVersionFromExecutable(pcx *process.ProcessContext) string {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return ""
	}

	file, err := elf.NewFile(exeFile)
	if err != nil {
		return ""
	}
	defer exeFile.Seek(0, io.SeekStart)

	for _, section := range file.Sections {
		// SHT_PROGBITS sections contain actual data: code (.text), read-only data (.rodata), writable data (.data), etc.
		// We want to read these sections when scanning for strings embedded in the binary.
		if section.Type != elf.SHT_PROGBITS {
			continue
		}

		data, err := section.Data()
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
