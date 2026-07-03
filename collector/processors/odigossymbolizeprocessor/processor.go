// Package odigossymbolizeprocessor is a profiles processor that names native
// (C/C++/Rust) frames the eBPF profiler left as module+offset. It does NOT analyze
// binaries itself: it batches each profile's native frames and asks the node-local
// symbolize server (run by vm-agent / odiglet, which has /proc access) to resolve
// them, then fills the OTLP Lines + tags them odigos.symbol.source. The heavy ELF
// work — and its memory/CPU peaks — lives in that separate process, keeping the
// collector's data path light. Frames the profiler already named (interpreted
// runtimes, Go) pass through untouched. See README.md.
package odigossymbolizeprocessor

import (
	"context"

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
}

func newProcessor(logger *zap.Logger, cfg *Config) *symbolizeProcessor {
	return &symbolizeProcessor{
		logger:   logger,
		cfg:      cfg,
		resolver: newResolver(cfg, logger),
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

	// Phase 1: collect every unresolved native frame across the batch (deduped by
	// location — the dictionary is shared, so a location is symbolized once).
	var pendingLocs []pprofile.Location
	var reqs []frameRequest
	seenLoc := make(map[int32]struct{})
	rps := pd.ResourceProfiles()
	for ri := 0; ri < rps.Len(); ri++ {
		rp := rps.At(ri)
		pid := pidFromResource(rp.Resource().Attributes(), p.cfg.pidAttribute())
		if pid <= 0 {
			continue
		}
		for locIdx := range reachableLocations(rp, stackTable) {
			if _, dup := seenLoc[locIdx]; dup {
				continue
			}
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
			seenLoc[locIdx] = struct{}{}
			pendingLocs = append(pendingLocs, loc)
			reqs = append(reqs, frameRequest{pid: pid, mod: m, addr: loc.Address()})
		}
	}
	if len(reqs) == 0 {
		return pd, nil
	}

	// Phase 2: one RPC to the symbolize server for the whole batch.
	results := p.resolver.resolveBatch(reqs)

	// Phase 3: fill the resolved names + tag native symbols (instrumentable).
	for i, res := range results {
		if !res.ok {
			continue
		}
		loc := pendingLocs[i]
		loc.Lines().AppendEmpty().SetFunctionIndex(internFunc(res.name))
		if res.source != "" {
			loc.AttributeIndices().Append(internSourceAttr(res.source))
		}
	}

	return pd, nil
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
