//go:build linux

// helpers_test.go holds shared test helpers: starting a Symbolizer, waiting for
// the background parse to complete, and building a real shared library to resolve.
package symbolize

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// newSym starts a Symbolizer and stops its workers at test end.
func newSym(t *testing.T, opts ...Option) *Symbolizer {
	t.Helper()
	s := New(opts...)
	t.Cleanup(s.Close)
	return s
}

// resolveWait polls Resolve until it succeeds or times out — symbol tables are
// parsed on a background worker, so resolution is eventually-consistent.
func resolveWait(t *testing.T, s *Symbolizer, pid int, m Mapping, addr uint64) (Frame, bool) {
	t.Helper()
	deadline := time.Now().Add(4 * time.Second)
	for {
		if f, ok := s.Resolve(pid, m, addr); ok {
			return f, true
		}
		if time.Now().After(deadline) {
			return Frame{}, false
		}
		time.Sleep(10 * time.Millisecond)
	}
}

// warm points a fake /proc at the given library for pid, then blocks until its
// symbols are parsed and cached (resolving with no build-id once parsed).
func warm(t *testing.T, s *Symbolizer, pid int, so string, addr uint64) {
	t.Helper()
	t.Setenv("HOST_PROC", writeFakeProc(t, pid, "0-1000 r-xp 0 08:01 1 "+so+"\n"))
	if _, ok := resolveWait(t, s, pid, Mapping{Name: filepath.Base(so)}, addr); !ok {
		t.Fatalf("failed to warm %s", so)
	}
}

// buildTestLib compiles a small C++ shared library (with a GNU build-id and an
// anonymous-namespace function — the hardest symbol-table case) and returns its
// path, the function's virtual address, and its build-id.
func buildTestLib(t *testing.T) (so string, addr uint64, buildID string) {
	t.Helper()
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ not available")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "lib.cpp")
	so = filepath.Join(dir, "libfix.so")
	const code = `
namespace { int worker(int n){ volatile int s=0; for(int i=0;i<n;i++) s+=i; return s; } }
extern "C" int entry(){ return worker(3); }
`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("g++", "-shared", "-fPIC", "-O0", "-g0", "-Wl,--build-id", "-o", so, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	es, err := loadELFSymbols(so, parseLimits{})
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range es.functions {
		if strings.Contains(f.name, "worker") {
			addr = f.addr
		}
	}
	if addr == 0 {
		t.Fatal("anonymous-namespace worker symbol not found")
	}
	return so, addr, es.buildID
}

// copyFile copies src to dst (used to make many distinct binaries for cache tests).
func copyFile(t *testing.T, src, dst string) {
	t.Helper()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dst, b, 0o755); err != nil {
		t.Fatal(err)
	}
}
