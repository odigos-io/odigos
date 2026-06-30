//go:build linux

package symbolize

// ladder.go extends symbol resolution beyond a binary's own .symtab/.dynsym to
// the forms stripped production binaries use: MiniDebugInfo (.gnu_debugdata,
// RHEL/Fedora) and separate debug files (.gnu_debuglink / build-id split-debug,
// Debian -dbgsym, RHEL debuginfo packages). All in-process — no external tools.
//
// Paths are container-aware: the collector reads a target's binaries through
// /proc/<pid>/root/..., so separate debug files must be resolved within that
// same root, not the host's filesystem.

import (
	"bytes"
	"debug/elf"
	"io"
	"os"
	"path/filepath"
	"regexp"

	"github.com/ulikunitz/xz"
)

// maxMiniDebugBytes bounds the decompressed MiniDebugInfo ELF.
const maxMiniDebugBytes = 256 << 20

// procRootRe matches the "/proc/<pid>/root" prefix of a container-rooted path.
var procRootRe = regexp.MustCompile(`^/proc/\d+/root`)

// symbolsFromMiniDebug extracts STT_FUNC symbols from the xz-compressed ELF
// embedded in .gnu_debugdata (MiniDebugInfo). Returns nil if absent/unreadable.
func symbolsFromMiniDebug(f *elf.File) []functionSymbol {
	sec := f.Section(".gnu_debugdata")
	if sec == nil {
		return nil
	}
	raw, err := sec.Data()
	if err != nil || len(raw) == 0 {
		return nil
	}
	r, err := xz.NewReader(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	decoded, err := io.ReadAll(io.LimitReader(r, maxMiniDebugBytes))
	if err != nil {
		return nil
	}
	inner, err := elf.NewFile(bytes.NewReader(decoded))
	if err != nil {
		return nil
	}
	defer func() { _ = inner.Close() }()
	// MiniDebugInfo carries .symtab only.
	return functionSymbolsFrom(inner.Symbols)
}

// symbolsFromDebugFile opens a separate debuginfo ELF and reads its symbols.
func symbolsFromDebugFile(path string, lim parseLimits) []functionSymbol {
	if lim.maxFileBytes > 0 {
		if fi, err := os.Stat(path); err == nil && fi.Size() > lim.maxFileBytes {
			return nil
		}
	}
	f, err := elf.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	syms, _ := readFunctionSymbols(f)
	return syms
}

// rootPrefix returns the container root of a /proc/<pid>/root/... path (so debug
// lookups stay inside the same mount namespace), or "" for a host-absolute path.
func rootPrefix(binPath string) string {
	return procRootRe.FindString(binPath)
}

// localDebugInfoPath finds an on-disk separate debug file for the binary at
// binPath, by build-id (<root>/usr/lib/debug/.build-id/ab/rest.debug) or by
// .gnu_debuglink (next to the binary, a .debug subdir, or under /usr/lib/debug).
// All candidates are resolved within binPath's container root.
func localDebugInfoPath(f *elf.File, binPath, buildID string) string {
	root := rootPrefix(binPath)
	if buildID != "" && len(buildID) > 2 {
		p := filepath.Join(root, "/usr/lib/debug/.build-id", buildID[:2], buildID[2:]+".debug")
		if statOK(p) {
			return p
		}
	}
	link := gnuDebugLink(f)
	if link == "" {
		return ""
	}
	dir := filepath.Dir(binPath)
	// dirInRoot is the binary's directory as seen inside the container root, used
	// to mirror it under /usr/lib/debug.
	dirInRoot := dir
	if root != "" {
		dirInRoot = dir[len(root):]
	}
	for _, cand := range []string{
		filepath.Join(dir, link),
		filepath.Join(dir, ".debug", link),
		filepath.Join(root, "/usr/lib/debug", dirInRoot, link),
	} {
		if statOK(cand) {
			return cand
		}
	}
	return ""
}

// gnuDebugLink returns the debug filename recorded in .gnu_debuglink, or "".
func gnuDebugLink(f *elf.File) string {
	sec := f.Section(".gnu_debuglink")
	if sec == nil {
		return ""
	}
	d, err := sec.Data()
	if err != nil {
		return ""
	}
	if i := bytes.IndexByte(d, 0); i >= 0 { // NUL-terminated filename, then CRC
		return string(d[:i])
	}
	return ""
}

func statOK(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && !fi.IsDir()
}
