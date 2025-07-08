package rust

import (
	"debug/elf"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RustInspector struct{}

func (n *RustInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *RustInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return "", false
	}

	file, err := elf.NewFile(exeFile)
	if err != nil {
		return "", false
	}

	// Check static symbols (from .symtab)
	staticSyms, err := file.Symbols()
	if err == nil {
		for _, sym := range staticSyms {
			if strings.Contains(sym.Name, "__rust_") {
				return common.RustProgrammingLanguage, true
			}
		}
	}

	// Check dynamic symbols (from .dynsym)
	dynSyms, err := file.DynamicSymbols()
	if err == nil {
		for _, sym := range dynSyms {
			if strings.Contains(sym.Name, "__rust_") {
				return common.RustProgrammingLanguage, true
			}
		}
	}

	return "", false
}

func (n *RustInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	// TODO: Implement this function to get the Rust runtime version
	return nil
}
