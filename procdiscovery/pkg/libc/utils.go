package libc

import (
	"debug/elf"
	"errors"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// ShouldInspectForLanguage returns true if we should inspect libc type for the given language.
//   - .NET: selects the glibc vs musl CoreCLR profiler variant (ModifyEnvVarsForMusl).
//   - C++/Rust: memory profiling LD_PRELOADs a glibc-built jemalloc; the instrumentor
//     gates that preload on a detected glibc libc, because musl's dynamic loader aborts
//     the process on an incompatible/missing preload (glibc's only warns). Without this
//     signal the gate fails closed and native-default profiling never engages. A
//     statically-linked native binary has no PT_INTERP, so InspectType returns nil and
//     the preload is correctly skipped (you cannot LD_PRELOAD a static binary anyway).
func ShouldInspectForLanguage(lang common.ProgrammingLanguage) bool {
	switch lang {
	case common.DotNetProgrammingLanguage,
		common.CPlusPlusProgrammingLanguage,
		common.RustProgrammingLanguage,
		// Interpreted runtimes are LD_PRELOAD'd with the libmemsample interposer for
		// memory profiling; the glibc-built lib ABORTS a musl (Alpine) process at load
		// (missing __snprintf_chk / ld-linux-x86-64.so.2). Detect libc so the
		// instrumentor preloads the musl-built variant into Alpine Python/Ruby/PHP
		// instead of crashing the app.
		common.PythonProgrammingLanguage,
		common.RubyProgrammingLanguage,
		common.PhpProgrammingLanguage:
		return true
	default:
		return false
	}
}

// ModifyEnvVarsForMusl modifies the environment variables for the given language if musl libc is detected
func ModifyEnvVarsForMusl(lang common.ProgrammingLanguage, envs map[string]string) map[string]string {
	if envs == nil {
		return nil
	}

	if !ShouldInspectForLanguage(lang) {
		return envs
	}

	if lang == common.DotNetProgrammingLanguage {
		val, ok := envs["CORECLR_PROFILER_PATH"]
		if ok {
			envs["CORECLR_PROFILER_PATH"] = strings.Replace(val, "linux-glibc", "linux-musl", 1)
		}
	}

	return envs
}

// InspectType inspects the given process for libc type
func InspectType(pd *process.Details) (*common.LibCType, error) {
	f, err := elf.Open(process.ProcFilePath(pd.ProcessID, "exe"))
	if err != nil {
		return nil, err
	}

	defer f.Close() // nolint:errcheck // we can't do anything if it fails
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}

		interp := make([]byte, prog.Filesz)
		_, err := prog.ReadAt(interp, 0)
		if err != nil {
			return nil, err
		}

		// Check the interpreter path
		interpPath := strings.Trim(string(interp), "\x00")
		if strings.Contains(interpPath, "musl") {
			musl := common.Musl
			return &musl, nil
		} else if strings.Contains(interpPath, "ld-linux") {
			glibc := common.Glibc
			return &glibc, nil
		}
		return nil, errors.New("unknown libc type")
	}

	return nil, errors.New("unknown libc type")
}
