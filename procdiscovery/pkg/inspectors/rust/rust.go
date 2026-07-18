package rust

import (
	"debug/elf"
	"io"
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

// rustcHashPattern is the longest possible literal matched by
// rustcCommitHashRe: "/rustc/" + 40 hex chars + "/".
const rustcHashPatternLen = len("/rustc/") + 40 + len("/")

// scanChunkSize bounds how much of a section is held in memory at once while
// scanning for an embedded rustc commit hash. ELF .rodata/.data.rel.ro
// sections can be very large (many MBs), and reading them in full via
// elf.Section.Data() causes memory spikes; scanning in bounded chunks keeps
// peak memory roughly constant regardless of section size.
const scanChunkSize = 64 * 1024

// extractRustcCommitHash searches raw section data for an embedded rustc
// commit hash. Returns "" if none is found.
func extractRustcCommitHash(data []byte) string {
	match := rustcCommitHashRe.FindSubmatch(data)
	if match == nil {
		return ""
	}
	return string(match[1])
}

// scanForRustcCommitHash streams r in bounded chunks looking for an embedded
// rustc commit hash, instead of reading the entire underlying section into
// memory. A small overlap is carried between chunks so a match that
// straddles a chunk boundary is not missed.
func scanForRustcCommitHash(r io.Reader) string {
	return scanForRustcCommitHashChunked(r, scanChunkSize)
}

// scanForRustcCommitHashChunked is the same as scanForRustcCommitHash but
// takes an explicit chunk size, so tests can exercise boundary-straddling
// matches with a small buffer.
func scanForRustcCommitHashChunked(r io.Reader, chunkSize int) string {
	overlap := rustcHashPatternLen - 1
	buf := make([]byte, overlap+chunkSize)
	carry := 0

	for {
		n, err := r.Read(buf[carry : carry+chunkSize])
		if n > 0 {
			window := buf[:carry+n]
			if hash := extractRustcCommitHash(window); hash != "" {
				return hash
			}

			// Keep the trailing bytes in case a match straddles this chunk
			// boundary and the next one.
			if len(window) > overlap {
				carry = copy(buf, window[len(window)-overlap:])
			} else {
				carry = copy(buf, window)
			}
		}
		if err != nil {
			return ""
		}
	}
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

		hash := scanForRustcCommitHash(section.Open())
		if hash == "" {
			continue
		}

		if version, ok := rustcHashToVersion[hash]; ok {
			return version
		}
		return hash
	}

	return ""
}
