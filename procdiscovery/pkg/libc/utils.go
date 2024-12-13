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

// InspectType inspects the given process for libc type
func InspectType(process *process.Details) (*common.LibCType, error) {
	f, err := elf.Open(fmt.Sprintf("/proc/%d/exe", process.ProcessID))
	if err != nil {
		return nil, err
	}

	defer f.Close()
	for _, prog := range f.Progs {
		if prog.Type == elf.PT_INTERP {
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
	}

	return nil, errors.New("unknown libc type")
}
