package rust

import (
	"debug/elf"
	"io"
	"strings"

	"github.com/hashicorp/go-version"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RustInspector struct{}

func (n *RustInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return "", false
	}
	defer func() {
		_, _ = exeFile.Seek(0, io.SeekStart)
	}()

	file, err := elf.NewFile(exeFile)
	if err != nil {
		return "", false
	}

	// Check static symbols (from .symtab)
	staticSyms, err := file.Symbols()
	if err == nil {
		found := scanSymbols(staticSyms)
		if found {
			return common.RustProgrammingLanguage, true
		}
	}

	// Check dynamic symbols (from .dynsym)
	dynSyms, err := file.DynamicSymbols()
	if err == nil {
		found := scanSymbols(dynSyms)
		if found {
			return common.RustProgrammingLanguage, true
		}
	}

	return "", false
}

func (n *RustInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (n *RustInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) *version.Version {
	// TODO: Implement this function to get the Rust runtime version

	return nil
}

func scanSymbols(syms []elf.Symbol) bool {
	for _, sym := range syms {
		if strings.Contains(sym.Name, "rust") || strings.HasPrefix(sym.Name, "_RNv") ||
			strings.Contains(sym.Name, "core::") || strings.Contains(sym.Name, "std::") {
			return true
		}
	}

	return false
}
