//go:build linux

// elf.go reads one binary's ELF symbols and load segments, and does the address
// math needed to turn a runtime instruction address into a named function:
// it parses the symbol table, finds the function that contains an address, and
// extracts the GNU build-id used to verify the file is the one that was mapped.
package symbolize

import (
	"debug/elf"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
)

// functionSymbol is one STT_FUNC entry: its virtual address, size and name.
type functionSymbol struct {
	addr uint64
	size uint64
	name string
}

// loadSegment is one PT_LOAD segment, used to translate between a file offset
// and a virtual address for both PIE/.so and non-PIE executables.
type loadSegment struct {
	off, vaddr, filesz uint64
}

// elfSymbols is the parsed, symbolization-ready view of one binary.
type elfSymbols struct {
	functions []functionSymbol // sorted by address
	segments  []loadSegment
	buildID   string // hex GNU build-id, "" if absent
	source    string // "symtab" | "dynsym" | ""
	heapBytes int64  // estimated memory this entry holds (for the byte-bounded cache)
}

// parseLimits guards against pathological/corrupt binaries that could exhaust
// memory or a parse worker. They are deliberately generous — real binaries
// (including Oracle's ~50 MB libclntsh) pass; only absurd inputs are rejected.
type parseLimits struct {
	maxFileBytes   int64 // skip ELF files larger than this
	maxSymbols     int   // skip binaries with more than this many function symbols
	maxSymtabBytes int64 // skip decoding a symbol table (+ its string table) larger than this
}

// limitExceededError is returned when a binary trips a parseLimit; the caller
// back-off-caches it (so it isn't retried every batch) and logs at debug.
type limitExceededError struct{ msg string }

func (e limitExceededError) Error() string { return e.msg }

// loadELFSymbols opens path and extracts its function symbols, PT_LOAD segments
// and GNU build-id. It prefers .symtab (covers static/internal functions) and
// falls back to .dynsym (exported only). It returns an elfSymbols even when no
// function symbols are present, so the build-id is still available for
// verification. lim bounds memory/CPU for outliers.
func loadELFSymbols(path string, lim parseLimits) (*elfSymbols, error) {
	if lim.maxFileBytes > 0 {
		if fi, err := os.Stat(path); err == nil && fi.Size() > lim.maxFileBytes {
			return nil, limitExceededError{fmt.Sprintf("symbolize: %s too large (%d bytes)", path, fi.Size())}
		}
	}
	f, err := elf.Open(path)
	if err != nil {
		return nil, fmt.Errorf("symbolize: open elf %s: %w", path, err)
	}
	defer func() { _ = f.Close() }()

	es := &elfSymbols{buildID: readBuildID(f)}
	for _, p := range f.Progs {
		if p.Type == elf.PT_LOAD && p.Filesz > 0 {
			es.segments = append(es.segments, loadSegment{off: p.Off, vaddr: p.Vaddr, filesz: p.Filesz})
		}
	}

	functions, source := readFunctionSymbols(f, lim)
	if lim.maxSymbols > 0 && len(functions) > lim.maxSymbols {
		return nil, limitExceededError{fmt.Sprintf("symbolize: %s has %d symbols (> %d)", path, len(functions), lim.maxSymbols)}
	}
	if len(functions) > 0 {
		es.functions, es.source = functions, source
		sort.Slice(es.functions, func(i, j int) bool { return es.functions[i].addr < es.functions[j].addr })
	}
	es.heapBytes = estimateHeapBytes(es)
	return es, nil
}

// estimateHeapBytes approximates the memory an elfSymbols holds, so the cache can
// evict by total memory rather than entry count (one Oracle binary can hold tens
// of MB of symbols). Per function: ~32 B of struct/string header + name bytes.
func estimateHeapBytes(es *elfSymbols) int64 {
	var b int64
	for i := range es.functions {
		b += 32 + int64(len(es.functions[i].name))
	}
	b += int64(len(es.segments)) * 24
	return b
}

// readFunctionSymbols returns the STT_FUNC symbols, preferring .symtab then
// falling back to .dynsym, with a tag naming the source table.
//
// A symbol table larger than lim.maxSymtabBytes is skipped WITHOUT decoding:
// Go's elf.File.Symbols() materialises the whole table (every symbol + the full
// string table) transiently before we filter to STT_FUNC, so on a huge unstripped
// binary that transient — not the retained cache — is what spikes RSS. Gating on
// section size caps it before the allocation happens; the frames stay
// module+offset, which the pipeline handles gracefully.
func readFunctionSymbols(f *elf.File, lim parseLimits) ([]functionSymbol, string) {
	if symtabWithinLimit(f, ".symtab", lim.maxSymtabBytes) {
		if syms := functionSymbolsFrom(f.Symbols); len(syms) > 0 {
			return syms, "symtab"
		}
	}
	if symtabWithinLimit(f, ".dynsym", lim.maxSymtabBytes) {
		if syms := functionSymbolsFrom(f.DynamicSymbols); len(syms) > 0 {
			return syms, "dynsym"
		}
	}
	return nil, ""
}

// symtabWithinLimit reports whether the named symbol table (plus its linked
// string table) is small enough to decode. An absent table, or limit<=0,
// returns true — there is either nothing to decode or no gate configured.
func symtabWithinLimit(f *elf.File, name string, limit int64) bool {
	if limit <= 0 {
		return true
	}
	sec := f.Section(name)
	if sec == nil {
		return true
	}
	total := int64(sec.Size)
	// elf.File.Symbols() also reads the linked string table (Link → .strtab/.dynstr);
	// account for it so the gate reflects the real transient allocation.
	if int(sec.Link) < len(f.Sections) {
		total += int64(f.Sections[sec.Link].Size)
	}
	return total <= limit
}

// functionSymbolsFrom keeps only the named, addressed STT_FUNC entries from one
// ELF symbol table.
func functionSymbolsFrom(read func() ([]elf.Symbol, error)) []functionSymbol {
	syms, err := read()
	if err != nil {
		return nil
	}
	out := make([]functionSymbol, 0, len(syms))
	for i := range syms {
		s := syms[i]
		if s.Value == 0 || s.Name == "" || elf.ST_TYPE(s.Info) != elf.STT_FUNC {
			continue
		}
		out = append(out, functionSymbol{addr: s.Value, size: s.Size, name: s.Name})
	}
	return out
}

// toVirtualAddr converts a runtime instruction address to an ELF virtual address
// via the containing PT_LOAD segment. memStart is the mapping's load address and
// fileOffset its file offset (both from the profile/maps). A zero memStart means
// the address is already a virtual address.
func (es *elfSymbols) toVirtualAddr(memStart, fileOffset, addr uint64) (uint64, bool) {
	if memStart == 0 {
		return addr, true
	}
	if addr < memStart {
		return 0, false
	}
	fileOff := addr - memStart + fileOffset
	for _, seg := range es.segments {
		if fileOff >= seg.off && fileOff < seg.off+seg.filesz {
			return fileOff - seg.off + seg.vaddr, true
		}
	}
	return 0, false
}

// toFileOffset converts a virtual address back to its file offset (the inverse
// PT_LOAD walk). For non-PIE EXEC binaries vaddr != offset. Returns false when no
// loadable segment contains the address.
func (es *elfSymbols) toFileOffset(vaddr uint64) (uint64, bool) {
	for _, seg := range es.segments {
		if vaddr >= seg.vaddr && vaddr < seg.vaddr+seg.filesz {
			return vaddr - seg.vaddr + seg.off, true
		}
	}
	return 0, false
}

// functionAt returns the function whose [addr, addr+size) range contains vaddr
// (the nearest preceding symbol when its size is unknown).
func (es *elfSymbols) functionAt(vaddr uint64) (functionSymbol, bool) {
	i := sort.Search(len(es.functions), func(i int) bool { return es.functions[i].addr > vaddr }) - 1
	if i < 0 {
		return functionSymbol{}, false
	}
	fn := es.functions[i]
	if fn.size > 0 && vaddr >= fn.addr+fn.size {
		return functionSymbol{}, false
	}
	return fn, true
}

// readBuildID returns the GNU build-id as a hex string, or "" if absent.
func readBuildID(f *elf.File) string {
	sec := f.Section(".note.gnu.build-id")
	if sec == nil {
		return ""
	}
	data, err := sec.Data()
	if err != nil {
		return ""
	}
	id, ok := parseGNUNote(data, f.ByteOrder)
	if !ok {
		return ""
	}
	return hex.EncodeToString(id)
}

// parseGNUNote extracts the descriptor of the first NT_GNU_BUILD_ID (type 3) note
// from a .note.gnu.build-id payload. The note header words are in the ELF's byte
// order, and the loop guards against overflow/corruption before slicing.
func parseGNUNote(b []byte, bo binary.ByteOrder) ([]byte, bool) {
	const noteHdr = 12 // namesz(4) + descsz(4) + type(4)
	for len(b) >= noteHdr {
		nameSz := int64(bo.Uint32(b[0:4]))
		descSz := int64(bo.Uint32(b[4:8]))
		noteType := bo.Uint32(b[8:12])
		off := int64(noteHdr) + align4(nameSz)
		end := off + descSz
		if off < noteHdr || descSz < 0 || end > int64(len(b)) {
			return nil, false
		}
		if noteType == 3 { // NT_GNU_BUILD_ID
			return b[off:end], true
		}
		next := end + align4(descSz)
		if next <= 0 || next > int64(len(b)) {
			return nil, false
		}
		b = b[next:]
	}
	return nil, false
}

// align4 rounds n up to a 4-byte boundary (ELF notes are 4-byte aligned).
func align4(n int64) int64 { return (n + 3) &^ 3 }
