package python

import (
	"debug/elf"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PythonInspector struct{}

const (
	libPythonStr = "libpython3"
)

// pythonExeRegex matches executable names that represent Python interpreters.
// It allows for the following formats:
//   - python           (generic python executable)
//   - python3          (major version specified)
//   - python311        (major and minor version without a dot)
//   - python3.12       (major and minor version with a dot)
//
// The pattern ensures that after the "python" prefix, only numeric versions (optionally with a single dot) are allowed.
var pythonExeRegex = regexp.MustCompile(`^python(\d+(\.\d+)?)?$`)

// exeVersionRegex extracts the version from exe paths like /usr/bin/python3.11 or /usr/bin/python3.11.2
var exeVersionRegex = regexp.MustCompile(`python(\d+\.\d+(?:\.\d+)?)`)

func (p *PythonInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	baseExe := filepath.Base(pcx.ExePath)

	if pythonExeRegex.MatchString(baseExe) {
		return common.PythonProgrammingLanguage, true
	}

	return "", false
}

func (p *PythonInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if p.isLibPythonLinked(pcx) {
		return common.PythonProgrammingLanguage, true
	}

	return "", false
}

func (p *PythonInspector) GetRuntimeVersion(proc *process.ProcessContext) string {
	// 1. PYTHON_VERSION env var (set by official Docker images)
	if value, exists := proc.GetDetailedEnvsValue(process.PythonVersionConst); exists {
		return value
	}

	// 2. Exe path e.g. /usr/bin/python3.11 (3.11) or /usr/bin/python3.11.2 (3.11.2)
	baseExe := filepath.Base(proc.ExePath)
	if subMatch := exeVersionRegex.FindStringSubmatch(baseExe); len(subMatch) > 1 {
		return subMatch[1]
	}
	return ""
}

func (p *PythonInspector) isLibPythonLinked(pcx *process.ProcessContext) bool {
	exeFile, err := pcx.GetExeFile()

	if err != nil {
		return false
	}

	elfFile, err := elf.NewFile(exeFile)
	if err != nil {
		return false
	}
	defer elfFile.Close() // nolint:errcheck // we can't do anything if it fails

	dynamicSection, err := elfFile.DynString(elf.DT_NEEDED)
	if err != nil {
		return false
	}

	for _, dep := range dynamicSection {
		if strings.Contains(dep, libPythonStr) {
			return true
		}
	}

	return false
}
