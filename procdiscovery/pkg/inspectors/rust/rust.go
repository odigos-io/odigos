package rust

import (
	"debug/elf"
	"regexp"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

type RustInspector struct{}

// rustcCommitHashRe matches the rustc commit hash embedded in the Rust
// standard library's own source paths, e.g.
// "/rustc/79e9716c980570bfd1f666e3b16ac583f0168962/library/std/src/panicking.rs".
// The standard library is compiled into virtually every Rust binary (it is
// statically linked by default), and its panic/debug machinery embeds these
// paths as string literals, so the hash survives even in release builds
// without debug symbols.
//
// Unlike Go (which embeds a full semantic version, see debug/buildinfo),
// rustc does not embed its own semver into the binary it produces, so the
// commit hash is the most reliable runtime identifier available without
// making an external call to resolve hash -> release version.
var rustcCommitHashRe = regexp.MustCompile(`/rustc/([0-9a-f]{40})/`)

// extractRustcCommitHash searches raw section data for an embedded rustc
// commit hash. Returns "" if none is found.
func extractRustcCommitHash(data []byte) string {
	match := rustcCommitHashRe.FindSubmatch(data)
	if match == nil {
		return ""
	}
	return string(match[1])
}

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

func (n *RustInspector) GetRuntimeVersion(pcx *process.ProcessContext) string {
	exeFile, err := pcx.GetExeFile()
	if err != nil {
		return ""
	}

	file, err := elf.NewFile(exeFile)
	if err != nil {
		return ""
	}

	for _, section := range file.Sections {
		// String literals live in allocated, non-executable data sections
		// (e.g. .rodata, .data.rel.ro). Skip .text/.bss/etc to avoid
		// unnecessary work scanning code or zero-initialized memory.
		if section.Flags&elf.SHF_ALLOC == 0 || section.Flags&elf.SHF_EXECINSTR != 0 {
			continue
		}

		data, err := section.Data()
		if err != nil {
			continue
		}

		if hash := extractRustcCommitHash(data); hash != "" {
			return hash
		}
	}

	return ""
}
