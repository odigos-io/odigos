package python

import (
	"debug/elf"
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

func (p *PythonInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := pcx.Details
	if strings.Contains(proc.ExePath, pythonProcessName) || strings.Contains(proc.CmdLine, pythonProcessName) {
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
func (p *PythonInspector) GetRuntimeVersion(proc *process.Details, containerURL string) *version.Version {
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
	defer elfFile.Close()

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
