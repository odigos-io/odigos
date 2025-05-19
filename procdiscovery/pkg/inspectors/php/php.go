package php

import (
	"bytes"
	"debug/elf"
	"regexp"

	"path/filepath"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/inspectors/utils"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PhpInspector struct{}

var (
	processNames  = []string{"php", "php-fpm"}
	versionRegex  = regexp.MustCompile(`X-Powered-By:\s*PHP/(\d+\.\d+\.\d+)`)
	versionPrefix = "X-Powered-By: PHP/"
)

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
	if value, exists := pcx.GetDetailedEnvsValue(process.PhpVersionConst); exists {
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
