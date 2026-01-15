package rust

import (
	"bytes"
	"debug/elf"
	"io"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RustInspector struct{}

func (r *RustInspector) QuickScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	return "", false
}

func (r *RustInspector) DeepScan(pcx *process.ProcessContext) (common.ProgrammingLanguage, bool) {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return "", false
	}

	file, err := elf.NewFile(exeFile)
	if err != nil {
		return "", false
	}

	if r.checkSymbols(file) {
		return common.RustProgrammingLanguage, true
	}

	if r.checkELFSections(file) {
		return common.RustProgrammingLanguage, true
	}

	if r.checkPanicStrings(exeFile) {
		return common.RustProgrammingLanguage, true
	}

	if r.checkRustLibraries(file) {
		return common.RustProgrammingLanguage, true
	}

	return "", false
}

func (r *RustInspector) checkSymbols(file *elf.File) bool {
	staticSyms, err := file.Symbols()
	if err == nil {
		for _, sym := range staticSyms {
			if isRustSymbol(sym.Name) {
				return true
			}
		}
	}

	dynSyms, err := file.DynamicSymbols()
	if err == nil {
		for _, sym := range dynSyms {
			if isRustSymbol(sym.Name) {
				return true
			}
		}
	}

	return false
}

func isRustSymbol(name string) bool {
	rustPrefixes := []string{
		"__rust_",
		"rust_begin_unwind",
		"rust_eh_personality",
	}

	for _, prefix := range rustPrefixes {
		if strings.Contains(name, prefix) {
			return true
		}
	}

	rustMangledPrefixes := []string{
		"_ZN4core",
		"_ZN5alloc",
		"_ZN3std",
	}

	for _, prefix := range rustMangledPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}

	return false
}

func (r *RustInspector) checkELFSections(file *elf.File) bool {
	for _, section := range file.Sections {
		if section.Name == ".rustc" || section.Name == ".note.rustc" {
			return true
		}
	}
	return false
}

func (r *RustInspector) checkPanicStrings(exeFile process.ProcessFile) bool {
	rustPanicStrings := []string{
		"rust_begin_unwind",
		"panicked at",
		"called `Option::unwrap()` on a `None` value",
		"called `Result::unwrap()` on an `Err` value",
		"/rustc/",
		".cargo/registry",
	}

	if _, err := exeFile.Seek(0, io.SeekStart); err != nil {
		return false
	}

	buf := make([]byte, 512*1024)
	for {
		n, err := exeFile.Read(buf)
		if err != nil && err != io.EOF {
			break
		}
		if n == 0 {
			break
		}

		chunk := buf[:n]
		for _, pattern := range rustPanicStrings {
			if bytes.Contains(chunk, []byte(pattern)) {
				return true
			}
		}

		if err == io.EOF {
			break
		}
	}

	return false
}

func (r *RustInspector) checkRustLibraries(file *elf.File) bool {
	libs, err := file.ImportedLibraries()
	if err != nil {
		return false
	}

	for _, lib := range libs {
		if strings.Contains(lib, "libstd-") && strings.Contains(lib, ".so") {
			return true
		}
	}

	return false
}

func (r *RustInspector) GetRuntimeVersion(pcx *process.ProcessContext, containerURL string) string {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return ""
	}

	if _, err := exeFile.Seek(0, io.SeekStart); err != nil {
		return ""
	}

	buf := make([]byte, 256*1024)
	n, err := exeFile.Read(buf)
	if err != nil && err != io.EOF {
		return ""
	}

	chunk := buf[:n]
	rustcVersionPrefix := []byte("/rustc/")
	if idx := bytes.Index(chunk, rustcVersionPrefix); idx != -1 {
		end := idx + len(rustcVersionPrefix) + 40
		if end > len(chunk) {
			end = len(chunk)
		}
		versionBytes := chunk[idx+len(rustcVersionPrefix) : end]
		if nullIdx := bytes.IndexByte(versionBytes, 0); nullIdx != -1 {
			versionBytes = versionBytes[:nullIdx]
		}
		version := string(versionBytes)
		if len(version) > 0 && len(version) <= 40 {
			return version
		}
	}

	return ""
}
