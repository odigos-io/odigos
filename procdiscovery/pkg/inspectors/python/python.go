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

func (p *PythonInspector) QuickScan(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	proc := ctx.Details
	if strings.Contains(proc.ExePath, pythonProcessName) || strings.Contains(proc.CmdLine, pythonProcessName) {
		return common.PythonProgrammingLanguage, true
	}
	return "", false
}

func (p *PythonInspector) DeepScan(ctx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	if p.isLibPythonLinked(ctx) {
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

func (p *PythonInspector) isLibPythonLinked(ctx *process.ProcessContext) bool {
	ctx.ExeContent()

	elfFile, err := elf.NewFile(ctx.ExeFileContent)
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
