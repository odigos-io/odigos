package python

import (
	"debug/elf"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type PythonInspector struct{}

const (
	pythonProcessName = "python"
	libPythonStr      = "libpython3"
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

func (p *PythonInspector) GetRuntimeVersion(proc *process.ProcessContext, containerURL string) *version.Version {
	if value, exists := proc.GetDetailedEnvsValue(process.PythonVersionConst); exists {
		return common.GetVersion(value)
	}

	return nil
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
