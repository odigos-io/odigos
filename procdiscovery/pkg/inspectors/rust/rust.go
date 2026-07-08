package rust

import (
	"bytes"
	"debug/elf"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RustInspector struct{}

// rodataScanLimit bounds how much .rodata we scan for the rustc panic-path
// fingerprint, so detection stays cheap on large binaries.
const rodataScanLimit = 4 << 20 // 4 MiB

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

	// 1) Symbol tables: the surest signal, but absent on stripped binaries.
	// __rust_alloc / __rust_dealloc are the global allocator shims every Rust
	// program emits.
	if symsContain(file, "__rust_") {
		return common.RustProgrammingLanguage, true
	}

	// 2) Strip-surviving fingerprints. Release Rust binaries are routinely
	// stripped (no .symtab), which previously made them fall through to the
	// broad C++ inspector. These two signals survive stripping:
	//   - .comment carries the compiler tag, e.g. "rustc 1.78.0 (9b00956e5 ...)".
	//   - .rodata carries panic/location strings rooted at "/rustc/<hash>/library/"
	//     and "cargo/registry/src/", baked in by std and the build.
	if sectionContains(file, ".comment", []byte("rustc")) {
		return common.RustProgrammingLanguage, true
	}
	for _, sec := range []string{".rodata", ".rodata.str1.1"} {
		if sectionContainsAny(file, sec, [][]byte{
			[]byte("/rustc/"),
			[]byte("cargo/registry/src"),
			[]byte("/library/std/src/"),
		}, rodataScanLimit) {
			return common.RustProgrammingLanguage, true
		}
	}

	return "", false
}

// symsContain reports whether any static or dynamic symbol name contains needle.
func symsContain(file *elf.File, needle string) bool {
	if syms, err := file.Symbols(); err == nil {
		for i := range syms {
			if strings.Contains(syms[i].Name, needle) {
				return true
			}
		}
	}
	if syms, err := file.DynamicSymbols(); err == nil {
		for i := range syms {
			if strings.Contains(syms[i].Name, needle) {
				return true
			}
		}
	}
	return false
}

// sectionContains reports whether the named ELF section contains needle.
func sectionContains(file *elf.File, name string, needle []byte) bool {
	return sectionContainsAny(file, name, [][]byte{needle}, 0)
}

// sectionContainsAny reports whether the named section contains any of needles.
// If limit > 0 only the first limit bytes are scanned, bounding cost on large
// .rodata.
func sectionContainsAny(file *elf.File, name string, needles [][]byte, limit int) bool {
	sec := file.Section(name)
	if sec == nil {
		return false
	}
	data, err := sec.Data()
	if err != nil {
		return false
	}
	if limit > 0 && len(data) > limit {
		data = data[:limit]
	}
	for _, needle := range needles {
		if bytes.Contains(data, needle) {
			return true
		}
	}
	return false
}

func (n *RustInspector) GetRuntimeVersion(pcx *process.ProcessContext) string {
	// TODO: Implement this function to get the Rust runtime version
	return ""
}
