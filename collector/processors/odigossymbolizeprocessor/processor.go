// Package odigossymbolizeprocessor is a profiles processor that names native
// (C/C++/Rust) frames the eBPF profiler left as module+offset, by resolving them
// on-host from /proc/<pid>/maps and the binary's ELF symbols. Frames the profiler
// already named (interpreted runtimes, Go) pass through untouched. See README.md.
//
// processor.go is the pipeline entry point: for each ResourceProfiles it reads
// the process.pid, pre-warms the symbol cache for newly-seen processes, and fills
// each native Location's Lines with the resolved function name.
package odigossymbolizeprocessor

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

const buildIDGNUKey = "process.executable.build_id.gnu"

// symbolSourceKey is the Location attribute we attach to a frame we symbolized,
// carrying the ELF table it came from ("symtab"/"dynsym"). Both mean the symbol
// is in the live binary, so downstream tooling (e.g. odictl custom-instrumentation
// authoring) can treat the frame as a native uprobe target by signature, distinct
// from a frame the profiler named (kernel/interpreted, no Location attribute).
const symbolSourceKey = "odigos.symbol.source"

// symbolizeProcessor fills the empty Lines of native OTLP profile Locations by
// resolving their addresses against the originating process's on-disk symbols
// (on-host). Frames already named by the profiler (interpreted runtimes + Go,
// which carry Lines) are passed through untouched — so this is correct for any
// language with zero extra config. The same processor runs in the k8s node
// collector and the VM agent collector.
type symbolizeProcessor struct {
	logger   *zap.Logger
	cfg      *Config
	resolver resolver

	mu   sync.Mutex
	seen map[int64]struct{} // pids we've pre-warmed
}

func newProcessor(logger *zap.Logger, cfg *Config) *symbolizeProcessor {
	return &symbolizeProcessor{
		logger:   logger,
		cfg:      cfg,
		resolver: newResolver(cfg, logger),
		seen:     make(map[int64]struct{}),
	}
}

func (p *symbolizeProcessor) Shutdown(context.Context) error {
	p.resolver.close()
	return nil
}

func (p *symbolizeProcessor) processProfiles(_ context.Context, pd pprofile.Profiles) (pprofile.Profiles, error) {
	dict := pd.Dictionary()
	stringTable := dict.StringTable()
	funcTable := dict.FunctionTable()
	locTable := dict.LocationTable()
	mapTable := dict.MappingTable()
	stackTable := dict.StackTable()
	attrTable := dict.AttributeTable()

	if locTable.Len() == 0 {
		return pd, nil
	}

	getStr := func(i int32) string {
		if int(i) >= 0 && int(i) < stringTable.Len() {
			return stringTable.At(int(i))
		}
		return ""
	}

	funcIndex := make(map[string]int32)
	internFunc := func(name string) int32 {
		if i, ok := funcIndex[name]; ok {
			return i
		}
		strIdx := int32(stringTable.Len())
		stringTable.Append(name)
		fn := funcTable.AppendEmpty()
		fn.SetNameStrindex(strIdx)
		idx := int32(funcTable.Len() - 1)
		funcIndex[name] = idx
		return idx
	}

	// internSourceAttr returns the AttributeTable index for the odigos.symbol.source=<src>
	// attribute, creating it (and its interned key) once per distinct source.
	sourceKeyIdx := int32(-1)
	sourceAttrIndex := make(map[string]int32)
	internSourceAttr := func(src string) int32 {
		if i, ok := sourceAttrIndex[src]; ok {
			return i
		}
		if sourceKeyIdx < 0 {
			sourceKeyIdx = int32(stringTable.Len())
			stringTable.Append(symbolSourceKey)
		}
		kv := attrTable.AppendEmpty()
		kv.SetKeyStrindex(sourceKeyIdx)
		kv.Value().SetStr(src)
		idx := int32(attrTable.Len() - 1)
		sourceAttrIndex[src] = idx
		return idx
	}

	rps := pd.ResourceProfiles()
	for ri := 0; ri < rps.Len(); ri++ {
		rp := rps.At(ri)
		pid := pidFromResource(rp.Resource().Attributes(), p.cfg.pidAttribute())
		if pid <= 0 {
			continue
		}
		p.prewarmOnce(pid)

		for locIdx := range reachableLocations(rp, stackTable) {
			if locIdx < 0 || int(locIdx) >= locTable.Len() {
				continue
			}
			loc := locTable.At(int(locIdx))
			if loc.Lines().Len() > 0 {
				continue // already named by the profiler (interpreted/Go) — pass through
			}
			m, ok := mappingRef(loc, mapTable, attrTable, getStr)
			if !ok {
				continue
			}
			name, source, ok := p.resolver.resolve(pid, m, loc.Address())
			if !ok {
				// Never drop a native frame. When no symbol resolves (stripped
				// binary, debug info not reachable, not-yet-parsed, build-id
				// mismatch) fall back to "module+0xoffset" so the sample
				// survives and renders meaningfully instead of as "?"/being
				// dropped by the backend's unsymbolized-native filter. The
				// Mapping still carries the GNU build-id, so a later
				// debuginfod/DWARF pass can replace this synthetic name with the
				// real one (the "+0x" shape marks it as re-symbolizable).
				name = syntheticName(m, loc.Address())
				if name == "" {
					continue
				}
				source = ""
			}
			loc.Lines().AppendEmpty().SetFunctionIndex(internFunc(name))
			if source != "" {
				// mark this as a native symbol from the live binary (instrumentable)
				loc.AttributeIndices().Append(internSourceAttr(source))
			}
		}
	}

	return pd, nil
}

// prewarmOnce asynchronously parses a newly-seen process's mapped binaries so
// subsequent batches hit a warm symbol cache (no hot-path parse).
func (p *symbolizeProcessor) prewarmOnce(pid int64) {
	p.mu.Lock()
	_, ok := p.seen[pid]
	if !ok {
		p.seen[pid] = struct{}{}
	}
	p.mu.Unlock()
	if !ok {
		p.resolver.prewarm(pid)
	}
}

func reachableLocations(rp pprofile.ResourceProfiles, stackTable pprofile.StackSlice) map[int32]struct{} {
	out := make(map[int32]struct{})
	sps := rp.ScopeProfiles()
	for si := 0; si < sps.Len(); si++ {
		profs := sps.At(si).Profiles()
		for pi := 0; pi < profs.Len(); pi++ {
			samples := profs.At(pi).Samples()
			for s := 0; s < samples.Len(); s++ {
				stackIdx := samples.At(s).StackIndex()
				if stackIdx < 0 || int(stackIdx) >= stackTable.Len() {
					continue
				}
				li := stackTable.At(int(stackIdx)).LocationIndices()
				for k := 0; k < li.Len(); k++ {
					out[li.At(k)] = struct{}{}
				}
			}
		}
	}
	return out
}

// syntheticName is the never-drop fallback for a native frame no symbol could
// name: "<module>+0x<file-offset>" (e.g. "libssl.so+0x3f12c"). The offset is the
// address normalized to a file offset (so it is stable across PIE load bias),
// matching how perf/pprof render unsymbolized native frames. Returns "" only when
// there is no usable module name, in which case the caller drops the frame.
func syntheticName(m moduleRef, addr uint64) string {
	base := filepath.Base(m.Name)
	if base == "" || base == "." || base == "/" {
		return ""
	}
	off := addr
	if m.MemoryStart != 0 && addr >= m.MemoryStart {
		off = addr - m.MemoryStart + m.FileOffset
	}
	return fmt.Sprintf("%s+0x%x", base, off)
}

// mappingRef builds the moduleRef for a Location, including the GNU build-id
// from the mapping's attributes (for strict on-disk verification).
func mappingRef(loc pprofile.Location, mapTable pprofile.MappingSlice, attrTable pprofile.KeyValueAndUnitSlice, getStr func(int32) string) (moduleRef, bool) {
	if loc.Address() == 0 {
		return moduleRef{}, false
	}
	mi := loc.MappingIndex()
	if mi < 0 || int(mi) >= mapTable.Len() {
		return moduleRef{}, false
	}
	m := mapTable.At(int(mi))
	name := getStr(m.FilenameStrindex())
	if name == "" {
		return moduleRef{}, false
	}
	return moduleRef{
		Name:        name,
		MemoryStart: m.MemoryStart(),
		FileOffset:  m.FileOffset(),
		BuildID:     buildIDFromMapping(m, attrTable, getStr),
	}, true
}

func buildIDFromMapping(m pprofile.Mapping, attrTable pprofile.KeyValueAndUnitSlice, getStr func(int32) string) string {
	ai := m.AttributeIndices()
	for k := 0; k < ai.Len(); k++ {
		idx := int(ai.At(k))
		if idx < 0 || idx >= attrTable.Len() {
			continue
		}
		kv := attrTable.At(idx)
		if getStr(kv.KeyStrindex()) == buildIDGNUKey {
			return kv.Value().Str()
		}
	}
	return ""
}

func pidFromResource(attrs pcommon.Map, key string) int64 {
	v, ok := attrs.Get(key)
	if !ok {
		return 0
	}
	switch v.Type() {
	case pcommon.ValueTypeInt:
		return v.Int()
	case pcommon.ValueTypeStr:
		n := int64(0)
		for _, c := range v.Str() {
			if c < '0' || c > '9' {
				return 0
			}
			n = n*10 + int64(c-'0')
		}
		return n
	default:
		return 0
	}
}
