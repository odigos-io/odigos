package php

import (
	"bytes"
	"debug/elf"
	"path/filepath"
	"regexp"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PhpInspector struct{}

var (
	// Matches "php", "php-cgi", "php-fpm", and versions like "php7.4", "php8.0", "php-fpm82"
	phpExecutableRegex = regexp.MustCompile("^php(-cgi|-fpm)?[0-9.]*$")
	versionRegex       = regexp.MustCompile(`X-Powered-By:\s*PHP/(\d+\.\d+\.\d+)`)
	versionPrefix      = "X-Powered-By: PHP/"
)

func (n *PhpInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if phpExecutableRegex.MatchString(filepath.Base(pcx.ExePath)) {
		return common.PhpProgrammingLanguage, true
	}

	return "", false
}

func (n *PhpInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *PhpInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	if value, exists := pcx.GetDetailedEnvsValue(process.PhpVersionConst); exists {
		return value
	}

	vers := getVersionFromExecutable(pcx)
	if vers != "" {
		return vers
	}

	return ""
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

	needle := []byte(versionPrefix)
	for _, section := range file.Sections {
		if section.Name != ".rodata" {
			continue
		}

		data, err := section.Data()
		if err != nil || len(data) == 0 {
			continue
		}

		idx := bytes.Index(data, needle)
		if idx < 0 {
			continue
		}
		idx += len(needle)

		zeroIdx := bytes.IndexByte(data[idx:], 0)
		if zeroIdx < 0 {
			continue
		}

		versionStr := string(data[idx : idx+zeroIdx])
		if matches := versionRegex.FindStringSubmatch(versionPrefix + versionStr); matches != nil {
			return versionStr
		}
	}

	return ""
}
