//go:build linux

package symbolize

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
)

// compileSO compiles src (C or C++) into a shared object with a GNU build-id and
// returns its path. The test is skipped when no toolchain is available.
func compileSO(t *testing.T, compiler, ext, src string) string {
	t.Helper()
	if _, err := exec.LookPath(compiler); err != nil {
		t.Skipf("%s not available", compiler)
	}
	dir := t.TempDir()
	srcPath := filepath.Join(dir, "fix"+ext)
	soPath := filepath.Join(dir, "libfix.so")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(compiler, "-shared", "-fPIC", "-O0", "-g0",
		"-Wl,--build-id", "-o", soPath, srcPath)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile failed: %v\n%s", err, out)
	}
	return soPath
}

// symbolVA returns the virtual address of a named symbol in path.
func symbolVA(t *testing.T, path, name string) uint64 {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	for _, read := range []func() ([]elf.Symbol, error){f.Symbols, f.DynamicSymbols} {
		syms, _ := read()
		for _, s := range syms {
			if s.Name == name {
				return s.Value
			}
		}
	}
	t.Fatalf("symbol %q not found in %s", name, path)
	return 0
}

// firstExecLoad returns the file offset and vaddr of the first executable
// PT_LOAD segment of path.
func firstExecLoad(t *testing.T, path string) (off, vaddr uint64) {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	for _, p := range f.Progs {
		if p.Type == elf.PT_LOAD && p.Flags&elf.PF_X != 0 {
			return p.Off, p.Vaddr
		}
	}
	t.Fatal("no executable PT_LOAD")
	return 0, 0
}

func TestLoadELFSymbols_C(t *testing.T) {
	so := compileSO(t, "cc", ".c", `
int compute_sum(int n) { int s = 0; for (int i = 0; i < n; i++) s += i; return s; }
`)
	es, err := loadELFSymbols(so, parseLimits{})
	if err != nil {
		t.Fatal(err)
	}
	if es.buildID == "" {
		t.Error("expected a GNU build-id")
	}
	if es.source != "symtab" && es.source != "dynsym" {
		t.Fatalf("source = %q", es.source)
	}
	// Look the symbol up by its own virtual address (memStart 0 => addr is vaddr).
	va := symbolVA(t, so, "compute_sum")
	fn, ok := es.functionAt(va)
	if !ok || fn.name != "compute_sum" {
		t.Fatalf("functionAt(%#x) = (%q,%v), want compute_sum", va, fn.name, ok)
	}
}

// TestResolveEndToEnd simulates the full path the vm-agent processor takes: a
// shared library loaded at an ASLR base in a process, found via /proc/<pid>/maps,
// symbolized from its on-disk ELF.
func TestResolveEndToEnd(t *testing.T) {
	so := compileSO(t, "g++", ".cpp", `
namespace soapcommand {
  struct Command { int onResume(); };
  int Command::onResume() { volatile int s = 0; for (int i=0;i<10;i++) s+=i; return s; }
}
extern "C" int entry() { soapcommand::Command c; return c.onResume(); }
`)

	const pid = 7777
	const loadBase = uint64(0x7f3300000000) // simulated ASLR load address

	off, vaddr := firstExecLoad(t, so)
	// Map the exec segment at loadBase with file offset == segment offset.
	mapStart := loadBase
	mapEnd := loadBase + 0x200000
	maps := "" +
		hexstr(mapStart) + "-" + hexstr(mapEnd) + " r-xp " + hexstr(off) +
		" 08:01 42 " + so + "\n"
	t.Setenv("HOST_PROC", writeFakeProc(t, pid, maps))

	// Pick a real C++ symbol and compute its runtime address under this mapping:
	//   addr = symVA - segVaddr + (mapStart) ... since fileOffset==segOff,
	//   toVirtualAddr recovers symVA. Use the mangled onResume symbol.
	name := mangledOnResume(t, so)
	symVA := symbolVA(t, so, name)
	addr := symVA - vaddr + mapStart

	s := newSym(t)
	frame, ok := resolveWait(t, s, pid, Mapping{
		Name:        "libfix.so",
		MemoryStart: mapStart,
		FileOffset:  off,
	}, addr)
	if !ok {
		t.Fatalf("Resolve failed for %#x", addr)
	}
	if !strings.Contains(frame.Name, "soapcommand::Command::onResume") {
		t.Fatalf("demangled name = %q, want it to contain soapcommand::Command::onResume", frame.Name)
	}
	if frame.Module != "libfix.so" {
		t.Errorf("module = %q", frame.Module)
	}

	// Build-id verification (cache is warm now): a wrong id must be refused; the real one accepted.
	if _, ok := s.Resolve(pid, Mapping{Name: "libfix.so", MemoryStart: mapStart, FileOffset: off, BuildID: "00bad00"}, addr); ok {
		t.Error("build-id mismatch must be refused")
	}
	es, _ := loadELFSymbols(so, parseLimits{})
	if _, ok := s.Resolve(pid, Mapping{Name: "libfix.so", MemoryStart: mapStart, FileOffset: off, BuildID: es.buildID}, addr); !ok {
		t.Error("matching build-id must be accepted")
	}
}

// mangledOnResume finds the mangled symbol for soapcommand::Command::onResume.
func mangledOnResume(t *testing.T, path string) string {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	for _, read := range []func() ([]elf.Symbol, error){f.Symbols, f.DynamicSymbols} {
		syms, _ := read()
		for _, s := range syms {
			if strings.Contains(s.Name, "onResume") && strings.HasPrefix(s.Name, "_Z") {
				return s.Name
			}
		}
	}
	t.Fatal("mangled onResume symbol not found")
	return ""
}

func hexstr(v uint64) string {
	const digits = "0123456789abcdef"
	if v == 0 {
		return "0"
	}
	var b [16]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = digits[v&0xf]
		v >>= 4
	}
	return string(b[i:])
}

func TestResolveUnknownModule(t *testing.T) {
	t.Setenv("HOST_PROC", writeFakeProc(t, 1, "00400000-00401000 r-xp 0 08:01 1 /bin/x\n"))
	s := newSym(t)
	if _, ok := s.Resolve(1, Mapping{Name: "not-mapped.so"}, 0x400100); ok {
		t.Fatal("Resolve should fail for an unmapped module")
	}
}

// TestCacheEviction checks the symbol cache stays within both its entry-count and
// byte budgets as many distinct binaries are resolved.
func TestCacheEviction(t *testing.T) {
	so, addr, buildID := buildTestLib(t)
	one, err := loadELFSymbols(so, parseLimits{})
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	// Byte budget sized for ~1.5 binaries forces eviction well before the count cap.
	s := newSym(t, WithMaxSymbolCache(1000), WithMaxSymbolBytes(one.heapBytes+one.heapBytes/2))

	for i := 0; i < 12; i++ {
		lib := filepath.Join(dir, fmt.Sprintf("lib%d.so", i))
		copyFile(t, so, lib)
		t.Setenv("HOST_PROC", writeFakeProc(t, 3000+i, "0-1000 r-xp 0 08:01 1 "+lib+"\n"))
		resolveWait(t, s, 3000+i, Mapping{Name: filepath.Base(lib), BuildID: buildID}, addr)
	}

	s.mu.Lock()
	bytes, count := s.cachedBytes, len(s.symbolCache)
	s.mu.Unlock()
	if bytes > s.maxSymbolBytes {
		t.Fatalf("cached bytes %d exceeded budget %d", bytes, s.maxSymbolBytes)
	}
	if count > 2 {
		t.Fatalf("byte budget should hold ~1 binary, got %d", count)
	}
}

// TestConcurrentResolve hits one Symbolizer from many goroutines once its cache
// is warm (run with -race).
func TestConcurrentResolve(t *testing.T) {
	so, addr, buildID := buildTestLib(t)
	const pid = 4242
	s := newSym(t)
	warm(t, s, pid, so, addr)

	const goroutines = 200
	var resolved, correct int64
	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			f, ok := s.Resolve(pid, Mapping{Name: filepath.Base(so), BuildID: buildID}, addr)
			if ok {
				atomic.AddInt64(&resolved, 1)
				if strings.Contains(f.Name, "worker") {
					atomic.AddInt64(&correct, 1)
				}
			}
		}()
	}
	wg.Wait()
	if resolved != goroutines || correct != goroutines {
		t.Fatalf("resolved=%d correct=%d, want %d/%d", resolved, correct, goroutines, goroutines)
	}
}
