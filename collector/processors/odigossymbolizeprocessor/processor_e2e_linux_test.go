//go:build linux

// processor_e2e_linux_test.go runs the full path end to end: a real compiled ELF
// and OTLP profile through processProfiles with the REAL symbolizer (not a fake),
// proving OTLP -> /proc lookup -> ELF resolution -> filled native Line connects.
package odigossymbolizeprocessor

import (
	"context"
	"debug/elf"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

// newProfilesWithNativeLoc builds a minimal OTLP profile with a single native
// location (no Line yet) at addr, mapped to module for the given pid.
func newProfilesWithNativeLoc(pid int, module string, addr uint64) pprofile.Profiles {
	pd := pprofile.NewProfiles()
	dict := pd.Dictionary()

	st := dict.StringTable()
	st.Append("")     // 0: sentinel
	st.Append(module) // 1: mapping filename

	mt := dict.MappingTable()
	m := mt.AppendEmpty()
	m.SetFilenameStrindex(1)
	m.SetMemoryStart(0) // VA-direct: location address is treated as a virtual address
	m.SetFileOffset(0)

	lt := dict.LocationTable()
	l0 := lt.AppendEmpty()
	l0.SetMappingIndex(0)
	l0.SetAddress(addr)

	stk := dict.StackTable()
	s := stk.AppendEmpty()
	s.LocationIndices().Append(0)

	rp := pd.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutInt("process.pid", int64(pid))
	prof := rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()
	sample := prof.Samples().AppendEmpty()
	sample.SetStackIndex(0)
	sample.Values().Append(1)
	return pd
}

// firstNativeLineName returns the resolved function name on location 0, or "".
func firstNativeLineName(pd pprofile.Profiles) string {
	dict := pd.Dictionary()
	lt := dict.LocationTable()
	if lt.Len() == 0 {
		return ""
	}
	loc := lt.At(0)
	if loc.Lines().Len() == 0 {
		return ""
	}
	fnIdx := loc.Lines().At(0).FunctionIndex()
	nameIdx := dict.FunctionTable().At(int(fnIdx)).NameStrindex()
	return dict.StringTable().At(int(nameIdx))
}

// buildE2ELib compiles a shared library exporting a C-named function so the
// symbol name is exactly what we assert (no C++ mangling).
func buildE2ELib(t *testing.T) (so, fn string) {
	t.Helper()
	if _, err := exec.LookPath("g++"); err != nil {
		t.Skip("g++ not available")
	}
	dir := t.TempDir()
	src := filepath.Join(dir, "e2e.cpp")
	so = filepath.Join(dir, "libe2e.so")
	fn = "e2e_entry"
	const code = `extern "C" int e2e_entry(int x){ volatile int s=0; for(int i=0;i<x;i++) s+=i; return s; }`
	if err := os.WriteFile(src, []byte(code), 0o644); err != nil {
		t.Fatal(err)
	}
	if out, err := exec.Command("g++", "-shared", "-fPIC", "-O0", "-Wl,--build-id", "-o", so, src).CombinedOutput(); err != nil {
		t.Fatalf("compile: %v\n%s", err, out)
	}
	return so, fn
}

// symbolVA returns the virtual address of a named symbol in an ELF.
func symbolVA(t *testing.T, path, want string) uint64 {
	t.Helper()
	f, err := elf.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	syms, err := f.Symbols()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range syms {
		if s.Name == want {
			return s.Value
		}
	}
	t.Fatalf("symbol %q not found in %s", want, path)
	return 0
}

// fakeProc writes a HOST_PROC tree so the symbolizer resolves the module
// basename to our real .so path (matched by name, not address).
func fakeProc(t *testing.T, pid int, so string) {
	t.Helper()
	root := t.TempDir()
	dir := filepath.Join(root, strconv.Itoa(pid))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	maps := "00000000-00001000 r-xp 00000000 08:01 1 " + so + "\n"
	if err := os.WriteFile(filepath.Join(dir, "maps"), []byte(maps), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOST_PROC", root)
}

// TestE2E_RealSymbolizerFillsNativeFrame drives the real symbolizer end to end.
func TestE2E_RealSymbolizerFillsNativeFrame(t *testing.T) {
	so, fnName := buildE2ELib(t)
	va := symbolVA(t, so, fnName)
	const pid = 7777
	fakeProc(t, pid, so)

	// MemoryStart=0 => location address is used directly as the symbol VA.
	pd := newProfilesWithNativeLoc(pid, "libe2e.so", va)

	p := newProcessor(zap.NewNop(), &Config{})
	defer func() { _ = p.Shutdown(context.Background()) }()

	// Parsing is async: prewarm then retry until the cache fills the Line.
	deadline := time.Now().Add(5 * time.Second)
	var gotName string
	for time.Now().Before(deadline) {
		out, err := p.processProfiles(context.Background(), pd)
		if err != nil {
			t.Fatalf("processProfiles: %v", err)
		}
		gotName = firstNativeLineName(out)
		if gotName == fnName {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if gotName != fnName {
		t.Fatalf("native frame not symbolized end to end: got %q, want %q", gotName, fnName)
	}
	t.Logf("E2E: OTLP native frame @%#x resolved through real symbolizer -> %q", va, gotName)
}
