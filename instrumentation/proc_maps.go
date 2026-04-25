package instrumentation

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
)

var procDir = func() string {
	if d := os.Getenv("ODIGOS_PROC_DIR"); d != "" {
		return d
	}
	return "/proc"
}()

// anyTriggerFileAlreadyMapped reports whether any of the given absolute paths
// is already present in /proc/<pid>/maps. False on any error (process gone,
// permission denied, no match) — caller falls back to waiting for FileOpen.
func anyTriggerFileAlreadyMapped(pid int, paths []string) bool {
	if pid <= 0 || len(paths) == 0 {
		return false
	}
	data, err := os.ReadFile(filepath.Join(procDir, strconv.Itoa(pid), "maps"))
	if err != nil {
		return false
	}
	// Each maps entry ends in " <pathname>\n"; matching that exact byte
	// sequence avoids false positives from offsets/inodes earlier in the
	// line and resolves to a single SIMD-accelerated bytes.Index per path.
	needle := make([]byte, 0, 256)
	for _, p := range paths {
		if p == "" {
			continue
		}
		needle = append(needle[:0], ' ')
		needle = append(needle, p...)
		needle = append(needle, '\n')
		if bytes.Contains(data, needle) {
			return true
		}
	}
	return false
}
