package symbolize

import (
	"bytes"
	"debug/elf"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/ianlancetaylor/demangle"
	"github.com/ulikunitz/xz"
)

// symbolizer resolves a native instruction address to the enclosing function,
// returning everything custom instrumentation needs (name, owning file, Build
// ID, file offset, size) plus where the symbol came from. The eBPF profiler
// symbolizes the kernel (kallsyms) and interpreted runtimes but leaves native
// userspace (C/C++/Rust) frames as mapping+address; vm-agent runs on the host
// where those binaries live, so we recover symbols locally.
//
// The whole ladder is gated by profiling.symbolization.native (off ⇒ resolve is
// a no-op, frames stay raw). All sources are local files already on the box —
// no network:
//
//  1. .symtab / .dynsym of the on-disk binary            (native)
//  2. .gnu_debugdata MiniDebugInfo (xz-compressed symtab)  (native)
//  3. local separate debuginfo by Build ID / debuglink    (native)
//
// Parsed results live in a process-global cache (globalSymbolCache) that survives
// across requests, so the expensive parse (symtab walk + xz MiniDebugInfo
// decompress) runs once per binary instead of on every ~2s poll. The symbolizer
// itself is cheap and per-request; only the feature gates are per-instance.
type symbolizer struct {
	native    bool // do on-disk symbolization at all (off ⇒ raw frames)
	instrMeta bool // compute uprobe offset + instrumentable verdict
}

// symCacheEntry is one binary's parsed symbols plus the stat stamp that validates
// the cache (an in-place redeploy at the same path changes modTime/size and forces
// a re-parse — correctness against stale-on-redeploy) and an LRU use-stamp. A nil
// es is a negative-cache entry: a parse that failed, not retried until the file
// changes.
type symCacheEntry struct {
	es      *elfSymbols
	modTime time.Time
	size    int64
	lastUse uint64
}

// symbolCache is the process-global, bounded LRU of parsed binaries shared by all
// symbolize requests. Keyed by resolved path (host path or /proc/<pid>/root/...);
// a per-entry stat stamp handles redeploys, and maxEntries bounds memory over the
// agent lifetime as the profiled binary set churns.
type symbolCache struct {
	mu         sync.Mutex
	entries    map[string]*symCacheEntry
	clock      uint64
	maxEntries int
}

// defaultSymbolCacheMaxEntries caps distinct parsed binaries kept at once. The set
// of binaries actively profiled on one VM is small; this is a safety valve against
// unbounded growth, not a tuning knob.
const defaultSymbolCacheMaxEntries = 128

var globalSymbolCache = &symbolCache{
	entries:    map[string]*symCacheEntry{},
	maxEntries: defaultSymbolCacheMaxEntries,
}

// get returns the parsed symbols for path, reusing the cached parse when the file's
// stat stamp is unchanged. On a miss or a stat change it calls parse (the expensive
// loader) and stores the result, evicting the least-recently-used entry past the cap.
func (sc *symbolCache) get(path string, parse func(string) *elfSymbols) *elfSymbols {
	var modTime time.Time
	var size int64
	if fi, err := os.Stat(path); err == nil {
		modTime = fi.ModTime()
		size = fi.Size()
	}

	sc.mu.Lock()
	if e, ok := sc.entries[path]; ok && e.modTime.Equal(modTime) && e.size == size {
		sc.clock++
		e.lastUse = sc.clock
		es := e.es
		sc.mu.Unlock()
		return es
	}
	sc.mu.Unlock()

	// Parse outside the lock — xz decompress + symtab walk can be slow, and we
	// don't want to block other binaries' lookups. A rare duplicate parse of the
	// same newly-seen binary under concurrency is harmless (last writer wins).
	es := parse(path)

	sc.mu.Lock()
	sc.clock++
	sc.entries[path] = &symCacheEntry{es: es, modTime: modTime, size: size, lastUse: sc.clock}
	for len(sc.entries) > sc.maxEntries {
		var oldestKey string
		var oldest uint64
		for k, e := range sc.entries {
			if oldestKey == "" || e.lastUse < oldest {
				oldest, oldestKey = e.lastUse, k
			}
		}
		if oldestKey == "" {
			break
		}
		delete(sc.entries, oldestKey)
	}
	sc.mu.Unlock()
	return es
}

// SymbolizeOptions carries the feature-gate down from the agent config without
// pkg/profilebundle importing components/config (avoids an import cycle).
type SymbolizeOptions struct {
	// Native enables the on-disk symbolization tiers (.symtab/.dynsym,
	// MiniDebugInfo, local debuginfo). Off ⇒ resolve() is a no-op and native
	// frames stay as the profiler's raw binary+address.
	Native bool
	// InstrumentationMetadata enables computing the uprobe-attach offset and the
	// instrumentable verdict for resolved symbols. Off ⇒ symbols are named for
	// display only (no probe/authoring surface).
	InstrumentationMetadata bool
}

func newSymbolizer(opts SymbolizeOptions) *symbolizer {
	return &symbolizer{
		native:    opts.Native,
		instrMeta: opts.InstrumentationMetadata,
	}
}

// ResolvedFrame is the instrumentation-ready result for one address.
type ResolvedFrame struct {
	Name    string // demangled; "" when unresolved
	Module  string // owning binary/.so basename
	BuildID string // hex GNU build-id of the resolving binary, "" if absent
	Offset  uint64 // file offset of the symbol entry — what a uprobe attaches at
	Size    uint64
	// Source: symtab|dynsym|minidebug|debuginfo|"". Whether the
	// symbol lives in the *live binary* (symtab/dynsym) vs only external
	// debuginfo decides whether OBI can attach today.
	Source string
}

// inLiveBinary reports whether the symbol is physically present in the running
// binary (so OBI's matcher can find it) vs recovered only from external/compressed
// debug data (which OBI does not read).
func (r ResolvedFrame) inLiveBinary() bool {
	return r.Source == "symtab" || r.Source == "dynsym"
}

type funcSym struct {
	addr uint64
	size uint64
	name string
}

type loadSeg struct {
	off, vaddr, filesz uint64
}

type elfSymbols struct {
	funcs   []funcSym // sorted by addr
	loads   []loadSeg // PT_LOAD segments, for addr→vaddr mapping
	buildID string
	source  string // where funcs came from for this file
}

// resolve maps a process instruction address (with the mapping's memory-start
// and file-offset) to a fully-described frame in exePath. ok is false when no
// symbol source could name the address.
func (s *symbolizer) resolve(exePath string, memStart, fileOffset, addr uint64) (ResolvedFrame, bool) {
	// Native symbolization gated off: behave as if no source could name the
	// address, so the frame renders as the profiler's raw binary+addr.
	if !s.native {
		return ResolvedFrame{Module: filepath.Base(exePath)}, false
	}
	es := s.symbolsFor(exePath)
	if es == nil || len(es.funcs) == 0 {
		return ResolvedFrame{Module: filepath.Base(exePath), BuildID: buildIDQuiet(es)}, false
	}
	vaddr, ok := es.toVaddr(memStart, fileOffset, addr)
	if !ok {
		return ResolvedFrame{Module: filepath.Base(exePath), BuildID: es.buildID}, false
	}
	fn, ok := es.lookup(vaddr)
	if !ok {
		return ResolvedFrame{Module: filepath.Base(exePath), BuildID: es.buildID}, false
	}
	rf := ResolvedFrame{
		Name:    demangle.Filter(fn.name),
		Module:  filepath.Base(exePath),
		BuildID: es.buildID,
		Size:    fn.size,
		Source:  es.source,
	}
	// Uprobe-attach offset is instrumentation metadata — compute only when that
	// gate is on. fn.addr is the symbol's virtual address; a uprobe attaches at
	// the file offset, which differs for non-PIE EXEC (translate via PT_LOAD,
	// fall back to vaddr). Display-only otherwise; OBI recomputes from Signature.
	if s.instrMeta {
		offset := fn.addr
		if off, ok := es.toFileOffset(fn.addr); ok {
			offset = off
		}
		rf.Offset = offset
	}
	return rf, true
}

func buildIDQuiet(es *elfSymbols) string {
	if es == nil {
		return ""
	}
	return es.buildID
}

func (s *symbolizer) symbolsFor(path string) *elfSymbols {
	// The parsed result is shared process-globally and validated by the file's
	// stat stamp, so repeated polls reuse the parse and a redeployed binary is
	// re-parsed. loadAllSources is gate-independent (it parses; the gates only
	// affect resolve()), so caching it across symbolizer instances is correct.
	return globalSymbolCache.get(path, s.loadAllSources)
}

// loadAllSources walks the symbol-source ladder for one file and returns the
// first source that yields function symbols. The on-disk binary is always read
// (for PT_LOAD mapping + Build ID); its symbols win when present, otherwise we
// fall through to MiniDebugInfo, then local separate debuginfo.
func (s *symbolizer) loadAllSources(path string) *elfSymbols {
	f, err := elf.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	es := &elfSymbols{buildID: elfBuildID(f)}
	for _, p := range f.Progs {
		if p.Type == elf.PT_LOAD && p.Filesz > 0 {
			es.loads = append(es.loads, loadSeg{off: p.Off, vaddr: p.Vaddr, filesz: p.Filesz})
		}
	}

	// 1. .symtab / .dynsym of the binary itself (passive).
	if syms, src := symbolsFromELF(f); len(syms) > 0 {
		es.funcs, es.source = syms, src
		finalizeSyms(es)
		return es
	}

	// 2. MiniDebugInfo: .gnu_debugdata = xz-compressed ELF with a .symtab (passive).
	if syms := symbolsFromMiniDebug(f); len(syms) > 0 {
		es.funcs, es.source = syms, "minidebug"
		finalizeSyms(es)
		return es
	}

	// 3. local separate debuginfo by Build ID / .gnu_debuglink (passive).
	if dbg := localDebugInfoPath(f, path, es.buildID); dbg != "" {
		if syms := symbolsFromFile(dbg); len(syms) > 0 {
			es.funcs, es.source = syms, "debuginfo"
			finalizeSyms(es)
			return es
		}
	}

	if len(es.funcs) == 0 {
		// Keep loads+buildID so callers can still report the module/build-id and
		// the toVaddr math; but no names available.
		return es
	}
	return es
}

func finalizeSyms(es *elfSymbols) {
	sort.Slice(es.funcs, func(i, j int) bool { return es.funcs[i].addr < es.funcs[j].addr })
}

// symbolsFromELF reads STT_FUNC entries from .symtab (preferred) then .dynsym.
// Returns the source label of whichever produced symbols.
func symbolsFromELF(f *elf.File) ([]funcSym, string) {
	if syms := funcSymsOf(f.Symbols); len(syms) > 0 {
		return syms, "symtab"
	}
	if syms := funcSymsOf(f.DynamicSymbols); len(syms) > 0 {
		return syms, "dynsym"
	}
	return nil, ""
}

func funcSymsOf(fn func() ([]elf.Symbol, error)) []funcSym {
	syms, err := fn()
	if err != nil {
		return nil
	}
	out := make([]funcSym, 0, len(syms))
	for i := range syms {
		s := syms[i]
		if s.Value == 0 || s.Name == "" || elf.ST_TYPE(s.Info) != elf.STT_FUNC {
			continue
		}
		out = append(out, funcSym{addr: s.Value, size: s.Size, name: stripSymVersion(s.Name)})
	}
	return out
}

// stripSymVersion removes the ELF symbol-versioning suffix ("name@@VERSION" for
// the default version, "name@VERSION" otherwise). The version is metadata from
// .gnu.version, not part of the mangled name; left attached it breaks demangling
// (the frame would surface as raw "_ZNSt...@@GLIBCXX_3.4") and any signature
// authored from it would never match OBI's demangled symbols.
func stripSymVersion(n string) string {
	if i := strings.IndexByte(n, '@'); i >= 0 {
		return n[:i]
	}
	return n
}

// symbolsFromMiniDebug extracts the .symtab from the xz-compressed ELF embedded
// in .gnu_debugdata (RHEL/Fedora MiniDebugInfo). Pure in-process; no commands.
func symbolsFromMiniDebug(f *elf.File) []funcSym {
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
	decoded, err := io.ReadAll(io.LimitReader(r, 256<<20))
	if err != nil {
		return nil
	}
	inner, err := elf.NewFile(bytes.NewReader(decoded))
	if err != nil {
		return nil
	}
	defer inner.Close()
	return funcSymsOf(inner.Symbols)
}

// symbolsFromFile opens a separate debuginfo ELF and reads its .symtab/.dynsym.
func symbolsFromFile(path string) []funcSym {
	f, err := elf.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	syms, _ := symbolsFromELF(f)
	return syms
}

// localDebugInfoPath finds an on-disk separate debuginfo file for f, by Build ID
// (/usr/lib/debug/.build-id/ab/rest.debug) or .gnu_debuglink next to the binary.
func localDebugInfoPath(f *elf.File, binPath, buildID string) string {
	if buildID != "" && len(buildID) > 2 {
		p := filepath.Join("/usr/lib/debug/.build-id", buildID[:2], buildID[2:]+".debug")
		if statOK(p) {
			return p
		}
	}
	if link := gnuDebugLink(f); link != "" {
		dir := filepath.Dir(binPath)
		for _, cand := range []string{
			filepath.Join(dir, link),
			filepath.Join(dir, ".debug", link),
			filepath.Join("/usr/lib/debug", dir, link),
		} {
			if statOK(cand) {
				return cand
			}
		}
	}
	return ""
}

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

// elfBuildID returns the GNU build-id as a hex string, or "".
func elfBuildID(f *elf.File) string {
	sec := f.Section(".note.gnu.build-id")
	if sec == nil {
		return ""
	}
	d, err := sec.Data()
	if err != nil || len(d) < 16 {
		return ""
	}
	// ELF note: namesz(4) descsz(4) type(4) name(namesz, padded) desc(descsz).
	nameSz := leU32(d[0:4])
	descSz := leU32(d[4:8])
	nameEnd := 12 + align4(int(nameSz))
	if nameEnd+int(descSz) > len(d) {
		return ""
	}
	return hex.EncodeToString(d[nameEnd : nameEnd+int(descSz)])
}

func leU32(b []byte) uint32 {
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
func align4(n int) int { return (n + 3) &^ 3 }

// toVaddr converts a process address to an ELF virtual address via the
// containing PT_LOAD segment. Handles non-PIE (EXEC) and PIE/.so. memStart==0
// means addr is already a vaddr.
func (es *elfSymbols) toVaddr(memStart, fileOffset, addr uint64) (uint64, bool) {
	if memStart == 0 {
		return addr, true
	}
	if addr < memStart {
		return 0, false
	}
	fileOff := addr - memStart + fileOffset
	for _, l := range es.loads {
		if fileOff >= l.off && fileOff < l.off+l.filesz {
			return fileOff - l.off + l.vaddr, true
		}
	}
	return 0, false
}

// toFileOffset converts a symbol's virtual address to its file offset — the
// inverse of toVaddr's PT_LOAD walk. A uprobe attaches at this file offset; for
// non-PIE EXEC vaddr != offset, so reporting the raw symbol value would be
// wrong. Returns false when no loadable segment contains the vaddr.
func (es *elfSymbols) toFileOffset(vaddr uint64) (uint64, bool) {
	for _, l := range es.loads {
		if vaddr >= l.vaddr && vaddr < l.vaddr+l.filesz {
			return vaddr - l.vaddr + l.off, true
		}
	}
	return 0, false
}

// lookup returns the function whose [addr, addr+size) contains vaddr (nearest
// preceding symbol when size is unknown).
func (es *elfSymbols) lookup(vaddr uint64) (funcSym, bool) {
	i := sort.Search(len(es.funcs), func(i int) bool { return es.funcs[i].addr > vaddr }) - 1
	if i < 0 {
		return funcSym{}, false
	}
	f := es.funcs[i]
	if f.size > 0 && vaddr >= f.addr+f.size {
		return funcSym{}, false
	}
	return f, true
}

func statOK(p string) bool { _, err := os.Stat(p); return err == nil }
