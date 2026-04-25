package instrumentation

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// anyTriggerFileAlreadyMapped reports whether any of the given absolute paths
// is already present in /proc/<pid>/maps. Used to short-circuit the
// FileOpen-trigger wait for processes that already have the trigger file
// loaded — typically (a) processes running before odiglet started (the
// runtime-detector's initial scan reports them as exec events even though
// the actual open() syscall happened long ago) and (b) workers forked from
// a parent that already opened the file (e.g. Python gunicorn in preload
// mode: the master loads the .so once, forked workers inherit the mapping
// and never call open() themselves, so no FileOpen event will ever fire
// for them).
//
// Returns false on any error (process gone, /proc not readable, no match)
// so that the caller falls back to waiting for a FileOpen event — that's
// the safe default; the worst that happens is a small delay.
func anyTriggerFileAlreadyMapped(pid int, paths []string) bool {
	if pid <= 0 || len(paths) == 0 {
		return false
	}
	procDir := os.Getenv("ODIGOS_PROC_DIR")
	if procDir == "" {
		procDir = "/proc"
	}
	f, err := os.Open(filepath.Join(procDir, strconv.Itoa(pid), "maps"))
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		// /proc/<pid>/maps lines end with the pathname column; matching on
		// " "+path avoids false positives from substrings appearing earlier
		// in the line (offsets, inodes, etc.).
		for _, p := range paths {
			if p != "" && strings.HasSuffix(line, " "+p) {
				return true
			}
		}
	}
	return false
}
