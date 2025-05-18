package libc

import (
	"debug/elf"
	"errors"
	"fmt"
	"strings"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/procdiscovery/pkg/process"
)

// ShouldInspectForLanguage returns true if we should inspect libc type for the given language
// Currently, we only inspect for .NET
func ShouldInspectForLanguage(lang common.ProgrammingLanguage) bool {
	return lang == common.DotNetProgrammingLanguage
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
	f, err := elf.Open(fmt.Sprintf("/proc/%d/exe", pd.ProcessID))
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
