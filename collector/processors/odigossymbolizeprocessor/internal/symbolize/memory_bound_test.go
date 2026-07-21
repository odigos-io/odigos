//go:build linux

// memory_bound_test.go proves the two memory guarantees behind CORE-1307:
//
//  1. the size-gate (maxSymtabBytes) skips a large symbol table WITHOUT
//     materialising it, so the transient decode that ballooned the Vodafone
//     collector's RSS (heap-inuse 66 MB while RSS hit 2.4 GB) never happens;
//  2. the byte-bounded symbol cache never holds more than its budget, however
//     many distinct binaries are symbolized.
//
// (1) is measured against a real unstripped native library (the Amdocs case),
// compiled at test time with the same g++ pattern the rest of the suite uses.
package symbolize

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// buildBigTestLib compiles a shared library with n trivial exported functions,
// so its .symtab (and .dynsym) is large — mimicking a big unstripped native .so
// (Oracle/Coherence/Amdocs) whose whole-table decode is the RSS spike we cap.
func buildBigTestLib(t *testing.T, n int) string {
	t.Helper()
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ not available")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "big.cpp")
	so := filepath.Join(dir, "libbig.so")

	var b strings.Builder
	for i := 0; i < n; i++ {
		fmt.Fprintf(&b, "extern \"C\" int fn_%d(int x){return x*%d+%d;}\n", i, i, i%7)
	}
	if err := os.WriteFile(src, []byte(b.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("g++", "-shared", "-fPIC", "-O0", "-g0", "-Wl,--build-id", "-o", so, src)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compile big lib: %v\n%s", err, out)
	}
	return so
}

// symtabSectionBytes returns the on-disk size of the binary's .symtab plus its
// linked string table — the bytes elf.File.Symbols() materialises transiently.
func symtabSectionBytes(t *testing.T, path string) int64 {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatalf("open elf: %v", err)
	}
	defer func() { _ = f.Close() }()
	sec := f.Section(".symtab")
	if sec == nil {
		return 0
	}
	total := int64(sec.Size)
	if int(sec.Link) < len(f.Sections) {
		total += int64(f.Sections[sec.Link].Size)
	}
	return total
}

// TestSizeGateEliminatesTransientDecode is the direct proof of the transient cap.
// We parse a real unstripped .so twice and compare bytes ALLOCATED (runtime
// TotalAlloc delta, which counts the transient []elf.Symbol + string table even
// though they're freed):
//   - gate OFF (maxSymtabBytes=0): stdlib decodes the whole table -> large alloc
//   - gate ON  (threshold below both tables): the tables are skipped -> tiny alloc
//
// The gated path must allocate dramatically less; we require >=4x to guard
// against silent regressions. The ratio grows with symtab size, so on a real
// 100+ MiB Oracle/Amdocs symtab the eliminated transient is hundreds of MB.
func TestSizeGateEliminatesTransientDecode(t *testing.T) {
	self := os.Getenv("SYMBOLIZE_TEST_ELF")
	if self == "" {
		self = buildBigTestLib(t, 8000) // ~200 KiB+ .symtab, ~seconds to compile
	}
	symBytes := symtabSectionBytes(t, self)
	if symBytes < 128<<10 {
		t.Skipf("fixture .symtab too small (%d bytes) to measure meaningfully", symBytes)
	}
	t.Logf(".symtab+.strtab on disk = %d bytes (%.2f MiB)", symBytes, float64(symBytes)/(1<<20))

	measure := func(lim parseLimits) uint64 {
		var m0, m1 runtime.MemStats
		runtime.GC()
		runtime.ReadMemStats(&m0)
		es, err := loadELFSymbols(self, lim)
		if err != nil {
			t.Fatalf("loadELFSymbols: %v", err)
		}
		runtime.ReadMemStats(&m1)
		runtime.KeepAlive(es)
		return m1.TotalAlloc - m0.TotalAlloc
	}

	base := parseLimits{maxFileBytes: defaultMaxFileBytes, maxSymbols: defaultMaxSymbols}

	off := base
	off.maxSymtabBytes = 0 // no gate: decode everything
	decodeAlloc := measure(off)

	// Gate below BOTH .symtab and .dynsym so neither table is decoded (an exported
	// .so carries the same functions in .dynsym; the fix code gates both tables).
	on := base
	on.maxSymtabBytes = 32 << 10 // 32 KiB, well under either table
	gatedAlloc := measure(on)

	// Sanity: prove we compared the two real paths — ungated must produce symbols,
	// gated must skip them.
	esOff, _ := loadELFSymbols(self, off)
	esOn, _ := loadELFSymbols(self, on)
	if len(esOff.functions) == 0 {
		t.Fatal("ungated parse produced no function symbols; fixture unusable")
	}
	if len(esOn.functions) != 0 {
		t.Fatalf("gated parse still decoded %d symbols; gate did not skip", len(esOn.functions))
	}

	ratio := float64(decodeAlloc) / float64(gatedAlloc+1)
	t.Logf("transient bytes allocated: decode=%d gated=%d (%.1fx less)", decodeAlloc, gatedAlloc, ratio)
	if gatedAlloc >= decodeAlloc {
		t.Fatalf("size-gate did not reduce allocation: decode=%d gated=%d", decodeAlloc, gatedAlloc)
	}
	if decodeAlloc < 4*gatedAlloc {
		t.Fatalf("size-gate reduction too small: decode=%d gated=%d (want decode >= 4x gated)", decodeAlloc, gatedAlloc)
	}
}

// TestSymbolCacheStaysWithinByteBudget proves the retained side stays bounded:
// however many distinct binaries are symbolized, the parsed-symbol cache never
// exceeds its configured byte budget. Fixture-free (synthetic entries) so it runs
// everywhere, not just where a compiler is present.
func TestSymbolCacheStaysWithinByteBudget(t *testing.T) {
	const entryBytes = 100_000
	budget := int64(4 * entryBytes) // holds only a few entries -> eviction must run
	s := New(WithMaxSymbolBytes(budget))
	defer s.Close()

	for i := 0; i < 512; i++ {
		e := &cachedSymbols{symbols: &elfSymbols{heapBytes: entryBytes}, heapBytes: entryBytes}
		e.lastUsed.Store(s.nextClock())
		s.mu.Lock()
		s.symbolCache[itoa(i)] = e
		s.cachedBytes += entryBytes
		s.evictSymbolsOverBudget()
		over := s.cachedBytes > s.maxSymbolBytes
		bytes := s.cachedBytes
		entries := len(s.symbolCache)
		s.mu.Unlock()
		if over {
			t.Fatalf("cachedBytes %d exceeded budget %d after %d inserts (entries=%d)", bytes, budget, i+1, entries)
		}
	}
	t.Logf("after 512 inserts of %d-byte entries under a %d-byte budget: cachedBytes=%d, entries=%d",
		entryBytes, budget, s.cachedBytes, len(s.symbolCache))
}
