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

var rustcCommitHashRe = regexp.MustCompile(`/rustc/([0-9a-f]{40})/`)

const rustcHashPatternLen = len("/rustc/") + 40 + len("/")

const scanChunkSize = 64 * 1024

func extractRustcCommitHash(data []byte) string {
	match := rustcCommitHashRe.FindSubmatch(data)
	if match == nil {
		return ""
	}
	return string(match[1])
}

func scanForRustcCommitHash(r io.Reader) string {
	return scanForRustcCommitHashChunked(r, scanChunkSize)
}

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

	staticSyms, err := file.Symbols()
	if err == nil {
		for _, sym := range staticSyms {
			if strings.Contains(sym.Name, "__rust_") {
				return common.RustProgrammingLanguage, true
			}
		}
	}

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
