package libc

import (
	"debug/elf"
	"errors"
	"os"
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

// InspectType inspects the given process for libc type.
//
// Primary signal is the executable's ELF PT_INTERP (the dynamic loader path).
// When that is unavailable — the exe is unreadable via /proc/<pid>/exe, or the
// PT_INTERP header cannot be read (observed for some dynamic glibc C++/Rust
// binaries) — we fall back to the process's mapped shared libraries in
// /proc/<pid>/maps, which carry a definitive libc signature. Without the
// fallback such a binary yields nil, the instrumentor's native-preload gate
// fails closed, and native memory profiling never engages.
func InspectType(pd *process.Details) (*common.LibCType, error) {
	if t := inspectFromInterp(pd); t != nil {
		return t, nil
	}
	if t := inspectFromMaps(pd); t != nil {
		return t, nil
	}
	return nil, errors.New("unknown libc type")
}

// inspectFromInterp reads the ELF PT_INTERP (dynamic loader) of the executable.
// Returns nil (not an error) when the exe/header can't be read or the interp
// path matches neither libc, so the caller can fall back to /proc/maps.
func inspectFromInterp(pd *process.Details) *common.LibCType {
	f, err := elf.Open(process.ProcFilePath(pd.ProcessID, "exe"))
	if err != nil {
		return nil
	}
	defer f.Close() // nolint:errcheck // we can't do anything if it fails
	for _, prog := range f.Progs {
		if prog.Type != elf.PT_INTERP {
			continue
		}
		interp := make([]byte, prog.Filesz)
		if _, err := prog.ReadAt(interp, 0); err != nil {
			return nil
		}
		interpPath := strings.Trim(string(interp), "\x00")
		switch {
		case strings.Contains(interpPath, "musl"):
			musl := common.Musl
			return &musl
		case strings.Contains(interpPath, "ld-linux"):
			glibc := common.Glibc
			return &glibc
		}
		return nil
	}
	return nil
}

// inspectFromMaps recognizes libc from the process's loaded shared libraries.
// musl is checked FIRST as a fail-safe: mistagging a musl process as glibc would
// let the instrumentor LD_PRELOAD a glibc-built lib, which musl's loader ABORTS
// (a crash); the reverse merely misses profiling. Returns nil for a truly static
// binary (no libc mapped) — correct, since you cannot LD_PRELOAD a static binary.
func inspectFromMaps(pd *process.Details) *common.LibCType {
	data, err := os.ReadFile(process.ProcFilePath(pd.ProcessID, "maps"))
	if err != nil {
		return nil
	}
	s := string(data)
	if strings.Contains(s, "ld-musl") || strings.Contains(s, "libc.musl") {
		musl := common.Musl
		return &musl
	}
	if strings.Contains(s, "libc.so.6") || strings.Contains(s, "/libc-") || strings.Contains(s, "ld-linux") {
		glibc := common.Glibc
		return &glibc
	}
	return nil
}
