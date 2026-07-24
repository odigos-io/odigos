//go:build linux

// memory_bound_test.go proves the size-gate skips a large symbol table without
// materialising it (bounding the transient decode) and the symbol cache stays
// within its byte budget, measured against a real g++-compiled native library.
package symbolize

import (
	"debug/elf"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
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

// TestSizeGateEliminatesTransientDecode proves the gate cuts the transient
// allocation: it parses the same binary with the gate off (decodes the whole
// table) and on (skips it), comparing bytes allocated via runtime.MemStats
// TotalAlloc, which counts memory freed after GC. Requires >=4x less to guard
// against silent regressions.
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

	// Below both .symtab and .dynsym, since an exported .so carries the same functions in .dynsym too.
	on := base
	on.maxSymtabBytes = 32 << 10
	gatedAlloc := measure(on)

	// Sanity: ungated must produce symbols, gated must skip them.
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

// TestConfigurableTransientCapIsHonored proves WithMaxSymtabBytes is wired: a
// small cap skips a binary whose symbol table exceeds it, while the default
// (512 MiB) decodes the same binary normally.
func TestConfigurableTransientCapIsHonored(t *testing.T) {
	so := buildBigTestLib(t, 8000) // ~200 KiB+ symbol table
	symBytes := symtabSectionBytes(t, so)
	if symBytes < 128<<10 {
		t.Skipf("fixture .symtab too small (%d bytes)", symBytes)
	}

	// Small cap (below the table): the binary is skipped without decoding, and
	// flagged as skipped-for-size (not just "no symbols") so the caller can warn.
	capped := New(WithMaxSymtabBytes(32 << 10))
	defer capped.Close()
	if es, err := loadELFSymbols(so, capped.limits); err != nil {
		t.Fatalf("capped parse: %v", err)
	} else if len(es.functions) != 0 {
		t.Fatalf("small transient cap must skip the table, got %d symbols", len(es.functions))
	} else if !es.symtabSkippedForSize {
		t.Fatal("expected symtabSkippedForSize=true when the table is skipped for exceeding the cap")
	}

	// Default cap (512 MiB): the same binary decodes normally, not flagged.
	def := New()
	defer def.Close()
	if es, err := loadELFSymbols(so, def.limits); err != nil {
		t.Fatalf("default parse: %v", err)
	} else if len(es.functions) == 0 {
		t.Fatal("default cap must decode a normal binary")
	} else if es.symtabSkippedForSize {
		t.Fatal("expected symtabSkippedForSize=false when the table decodes normally")
	}
}

// TestSymbolCacheStaysWithinByteBudget proves the retained cache never exceeds
// its configured byte budget, however many binaries are symbolized. Uses
// synthetic entries so it runs without a compiler.
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

// TestSymtabSkipsAreAggregatedNotPerBinary proves the noise-reduction fix: N
// distinct binaries tripping the max_symtab_bytes gate produce zero Warn logs
// from parseAndCache itself (only Debug, one per binary), and a single sweep
// drains the count into exactly one Warn — so a busy node with many oversized
// libraries, or one restarting across many pods, can't turn into a
// per-binary/per-restart warning storm.
func TestSymtabSkipsAreAggregatedNotPerBinary(t *testing.T) {
	so := buildBigTestLib(t, 8000)
	symBytes := symtabSectionBytes(t, so)
	if symBytes < 128<<10 {
		t.Skipf("fixture .symtab too small (%d bytes)", symBytes)
	}

	// Copy the same oversized binary to N distinct paths: parseAndCache dedups by
	// path, so this simulates N distinct binaries (or the same one across N
	// container instances) all tripping the gate.
	const n = 3
	dir := t.TempDir()
	paths := make([]string, n)
	data, err := os.ReadFile(so)
	if err != nil {
		t.Fatal(err)
	}
	for i := range paths {
		p := filepath.Join(dir, "lib"+strconv.Itoa(i)+".so")
		if err := os.WriteFile(p, data, 0o755); err != nil {
			t.Fatal(err)
		}
		paths[i] = p
	}

	core, logs := observer.New(zap.DebugLevel)
	s := New(WithMaxSymtabBytes(32<<10), WithLogger(zap.New(core)))
	defer s.Close()

	for _, p := range paths {
		s.parseAndCache(p)
	}

	if got := s.symtabSkips.Load(); got != n {
		t.Fatalf("symtabSkips = %d, want %d after %d skipped binaries", got, n, n)
	}
	if warns := logs.FilterLevelExact(zapcore.WarnLevel).Len(); warns != 0 {
		t.Fatalf("parseAndCache must not log Warn per binary, got %d Warn entries", warns)
	}
	skipDebugs := logs.FilterMessage("symbolize: symbol table exceeds max_symtab_bytes, skipping decode; native frames for this binary will stay unresolved")
	if got := skipDebugs.FilterLevelExact(zapcore.DebugLevel).Len(); got != n {
		t.Fatalf("expected %d per-binary skip Debug entries, got %d", n, got)
	}

	// One sweep drains the counter into exactly one Warn carrying the count.
	s.reportSymtabSkips()
	warnEntries := logs.FilterLevelExact(zapcore.WarnLevel).All()
	if len(warnEntries) != 1 {
		t.Fatalf("expected exactly 1 Warn after the sweep, got %d", len(warnEntries))
	}
	if got := warnEntries[0].ContextMap()["count"]; got != int64(n) {
		t.Fatalf("summary Warn count = %v, want %d", got, n)
	}

	// A second sweep with nothing new skipped must log nothing further.
	s.reportSymtabSkips()
	if warns := logs.FilterLevelExact(zapcore.WarnLevel).Len(); warns != 1 {
		t.Fatalf("expected still exactly 1 Warn total after an empty sweep, got %d", warns)
	}
}
