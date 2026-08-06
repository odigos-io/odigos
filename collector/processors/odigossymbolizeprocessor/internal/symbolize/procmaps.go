//go:build linux

// procmaps.go recovers, for one process, the openable on-disk path of each
// loaded module. The profiler ships only a mapping's basename, so we read
// /proc/<pid>/maps and index the file-backed executable mappings by basename,
// resolving each to a path the agent can open (container-aware).
package symbolize

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// procRoot is the procfs mount point. It honors HOST_PROC so an agent running
// inside a container (with the host's /proc bind-mounted elsewhere) can still
// read the maps of host processes. Matches the convention used across odigos.
func procRoot() string {
	if p := os.Getenv("HOST_PROC"); p != "" {
		return p
	}
	return "/proc"
}

// procMaps is a parsed /proc/<pid>/maps: a basename -> openable host path index
// over the process's file-backed, executable mappings.
type procMaps struct {
	pid        int
	pathByName map[string]string // module basename -> host path
}

// parseProcMaps reads /proc/<pid>/maps and indexes every file-backed executable
// mapping by basename. The first occurrence of a basename wins (the primary load
// segment is listed before later r-x slices of the same file).
func parseProcMaps(pid int) (*procMaps, error) {
	root := procRoot()
	f, err := os.Open(fmt.Sprintf("%s/%d/maps", root, pid))
	if err != nil {
		return nil, fmt.Errorf("symbolize: open maps for pid %d: %w", pid, err)
	}
	defer func() { _ = f.Close() }()

	pm := &procMaps{pid: pid, pathByName: make(map[string]string)}
	processRoot := fmt.Sprintf("%s/%d/root", root, pid)

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1<<20)
	for sc.Scan() {
		path, perms, ok := parseMapLine(sc.Text())
		if !ok {
			continue
		}
		if !strings.Contains(perms, "x") { // only executable mappings carry code we symbolize
			continue
		}
		if path == "" || path[0] != '/' || strings.HasPrefix(path, "/dev/") {
			continue
		}
		if strings.HasSuffix(path, " (deleted)") { // unlinked mid-run (redeploy): don't symbolize a stale file
			continue
		}
		name := filepath.Base(path)
		if _, seen := pm.pathByName[name]; seen {
			continue
		}
		pm.pathByName[name] = resolveHostPath(path, processRoot)
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("symbolize: scan maps for pid %d: %w", pid, err)
	}
	return pm, nil
}

// resolveHostPath returns a path the agent can open. /proc/<pid>/root/<path> is
// the one location reliable for EVERY process — host or containerized — because
// the kernel resolves it through that process's mount namespace. We prefer it
// over the bare host path so we never accidentally open a different file that
// happens to share the path in the agent's own namespace; the bare path is only
// a fallback when the namespaced one isn't statable.
func resolveHostPath(path, processRoot string) string {
	if namespaced := processRoot + path; fileExists(namespaced) {
		return namespaced
	}
	if fileExists(path) {
		return path
	}
	return processRoot + path // return it anyway; opening surfaces a precise error
}

// hostPath returns the openable host path for a module, given either its basename
// or a full path (whose basename is matched against the index).
func (pm *procMaps) hostPath(nameOrPath string) (string, bool) {
	p, ok := pm.pathByName[filepath.Base(nameOrPath)]
	return p, ok
}

// executablePaths returns the distinct openable host paths of this process's
// executable mappings (used by PreWarm).
func (pm *procMaps) executablePaths() []string {
	out := make([]string, 0, len(pm.pathByName))
	for _, p := range pm.pathByName {
		out = append(out, p)
	}
	return out
}

// processExists reports whether /proc/<pid> still exists (the process is alive).
// Used by the sweeper; a stat is cheap and reliable.
func processExists(pid int) bool {
	_, err := os.Stat(fmt.Sprintf("%s/%d", procRoot(), pid))
	return err == nil
}

// parseMapLine extracts the pathname and permissions from one /proc maps line.
// Format: "addr-addr perms offset dev inode pathname". The pathname may contain
// spaces, so the line is split into at most 6 fields and the rest taken verbatim.
func parseMapLine(line string) (path, perms string, ok bool) {
	fields := strings.SplitN(line, " ", 6)
	if len(fields) < 6 {
		return "", "", false
	}
	perms = fields[1]
	path = strings.TrimSpace(fields[5])
	if path == "" {
		return "", "", false
	}
	return path, perms, true
}

// fileExists reports whether path exists and is statable.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
