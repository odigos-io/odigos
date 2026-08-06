//go:build linux

package symbolize

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeProc lays out a HOST_PROC tree for pid with the given maps content
// and returns the root to set as HOST_PROC.
func writeFakeProc(t *testing.T, pid int, maps string) string {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "maps"), []byte(maps), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}

func TestParseProcMapsIndexesExecMappingsByBasename(t *testing.T) {
	// A realistic maps excerpt: a main exe, a shared lib (r-x and r--),
	// an anonymous mapping, and a deleted file — only the exec, real-file
	// entries should be indexed, first occurrence per basename winning.
	maps := "" +
		"00400000-004a0000 r-xp 00000000 08:01 100 /opt/app/server\n" +
		"7f0000000000-7f0000100000 r-xp 00000000 08:01 200 /lib/x86_64-linux-gnu/libc.so.6\n" +
		"7f0000100000-7f0000110000 r--p 00100000 08:01 200 /lib/x86_64-linux-gnu/libc.so.6\n" +
		"7f0000200000-7f0000300000 r-xp 00000000 08:01 300 /opt/app/libplugin.so (deleted)\n" +
		"7f0000400000-7f0000500000 rw-p 00000000 00:00 0 \n" +
		"7f0000600000-7f0000700000 r-xp 00000000 00:00 0 [vdso]\n"

	t.Setenv("HOST_PROC", writeFakeProc(t, 4242, maps))

	pm, err := parseProcMaps(4242)
	if err != nil {
		t.Fatal(err)
	}

	// The fake paths don't exist on the test host, so resolveHostPath falls back
	// to the /proc/<pid>/root-prefixed form; assert on the suffix, which is what
	// matters (the right module was indexed under its basename).
	if p, ok := pm.hostPath("server"); !ok || !strings.HasSuffix(p, "/opt/app/server") {
		t.Errorf("server: got (%q,%v), want suffix /opt/app/server", p, ok)
	}
	if p, ok := pm.hostPath("libc.so.6"); !ok || !strings.HasSuffix(p, "/lib/x86_64-linux-gnu/libc.so.6") {
		t.Errorf("libc: got (%q,%v)", p, ok)
	}
	// basename match works when caller passes a full container-side path too.
	if _, ok := pm.hostPath("/some/where/server"); !ok {
		t.Error("lookup by full path should match on basename")
	}
	// Deleted, non-exec, and special mappings must be excluded.
	if _, ok := pm.hostPath("libplugin.so"); ok {
		t.Error("deleted mapping must not be indexed")
	}
	if _, ok := pm.hostPath("[vdso]"); ok {
		t.Error("vdso must not be indexed")
	}
}

func TestResolveHostPathContainerFallback(t *testing.T) {
	root := t.TempDir()
	// The container sees /app/bin/svc; on the host it only exists under
	// /proc/<pid>/root. Simulate that layout.
	procRoot := filepath.Join(root, "5", "root")
	containerPath := "/app/bin/svc"
	hostSide := filepath.Join(procRoot, "app", "bin")
	if err := os.MkdirAll(hostSide, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(hostSide, "svc"), []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}

	got := resolveHostPath(containerPath, procRoot)
	want := procRoot + containerPath
	if got != want {
		t.Errorf("container fallback: got %q want %q", got, want)
	}
}

func TestParseMapLine(t *testing.T) {
	path, perms, ok := parseMapLine("00400000-004a0000 r-xp 00000000 08:01 100 /opt/a b/server")
	if !ok || perms != "r-xp" || path != "/opt/a b/server" {
		t.Errorf("got (%q,%q,%v) — path with space must be preserved", path, perms, ok)
	}
	if _, _, ok := parseMapLine("garbage"); ok {
		t.Error("malformed line should not parse")
	}
}
