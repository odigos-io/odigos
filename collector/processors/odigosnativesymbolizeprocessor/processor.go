package odigosnativesymbolizeprocessor

import (
	"context"
	"os"
	"path/filepath"
	"strconv"

	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/collector/processors/odigosnativesymbolizeprocessor/internal/symbolize"
)

// Resource attribute keys, sourced from semconv so they stay aligned with the eBPF profiler
// receiver, which emits these same keys.
const (
	attrProcessExecutablePath = string(semconv.ProcessExecutablePathKey)
	attrProcessPID            = string(semconv.ProcessPIDKey)
)

// nativeSymbolizeProcessor symbolizes native (C/C++/Rust) profile frames in-place using the
// on-disk binaries, then forwards the batch. Go/Java frames already carry Lines and pass
// through untouched. Symbolization is best-effort: if a binary cannot be found or an address
// cannot be resolved, the frame is left raw — the batch is never errored.
type nativeSymbolizeProcessor struct {
	logger *zap.Logger
	cfg    *Config
	sym    *symbolize.Symbolizer
}

// capabilities reports that this processor mutates pprofile data.
func (p *nativeSymbolizeProcessor) capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

// processProfiles symbolizes native frames in the batch and returns it for forwarding.
func (p *nativeSymbolizeProcessor) processProfiles(_ context.Context, profiles pprofile.Profiles) (pprofile.Profiles, error) {
	p.symbolizeNative(profiles)
	return profiles, nil
}

// symbolizeNative walks the profiles dictionary and, for each location that has no Lines
// (native, unsymbolized) but a mapping+address we can resolve to an on-disk binary, appends a
// Function + Line so downstream sees a function name instead of mapping+addr.
//
// The on-disk binary path for each mapping basename is resolved by binaryResolver, built from the
// batch's ResourceProfiles attributes (process.executable.path / process.pid). The
// location/mapping/string tables live in the batch-global dictionary (not per-resource), so a
// single resolver covering every resource is built once per batch. Locations whose binary cannot
// be found are left raw — symbolization is best-effort and never errors the batch.
func (p *nativeSymbolizeProcessor) symbolizeNative(profiles pprofile.Profiles) int {
	dict := profiles.Dictionary()
	strTab := dict.StringTable()
	funcTab := dict.FunctionTable()
	locTab := dict.LocationTable()
	mapTab := dict.MappingTable()

	resolver := newBinaryResolver(profiles.ResourceProfiles())
	if resolver.empty() {
		return 0
	}

	getStr := func(i int32) string {
		if i < 0 || int(i) >= strTab.Len() {
			return ""
		}
		return strTab.At(int(i))
	}

	count := 0
	for i := 0; i < locTab.Len(); i++ {
		loc := locTab.At(i)
		if loc.Lines().Len() > 0 {
			continue // already symbolized (Go/interpreted)
		}
		mi := loc.MappingIndex()
		if mi < 0 || int(mi) >= mapTab.Len() || loc.Address() == 0 {
			continue
		}
		m := mapTab.At(int(mi))
		fname := getStr(m.FilenameStrindex())
		if fname == "" {
			continue
		}
		exe := resolver.resolve(fname)
		if exe == "" {
			continue // binary not found for this mapping — leave the frame raw
		}
		rf, ok := p.sym.Resolve(exe, m.MemoryStart(), m.FileOffset(), loc.Address())
		if !ok || rf.Name == "" {
			continue
		}
		// append name -> StringTable, Function, Line
		nameIdx := strTab.Len()
		strTab.Append(rf.Name)
		fn := funcTab.AppendEmpty()
		fn.SetNameStrindex(int32(nameIdx))
		fnIdx := funcTab.Len() - 1
		ln := loc.Lines().AppendEmpty()
		ln.SetFunctionIndex(int32(fnIdx))
		count++
	}
	return count
}

// binaryResolver maps a mapping filename (from the OTLP string table) to an on-disk binary path,
// using the resource attributes of the batch's ResourceProfiles. It implements production binary
// path resolution (not the demo seed map):
//
//  1. exact basename → process.executable.path, when that resource attribute is present (the
//     authoritative path for the workload's main executable);
//  2. otherwise, for any mapping basename, /proc/<process.pid>/root/<basename> — the process root
//     mount, which exposes the same on-disk binaries (including shared libraries like libc.so.6)
//     for containerized workloads where only process.pid is known.
//
// A batch may carry multiple ResourceProfiles (multiple processes/containers). We collect each
// resource's executable path and PID; resolve() tries the basename→exe map first, then each PID
// root in turn, returning the first path that exists on disk.
type binaryResolver struct {
	exeIndex  map[string]string // basename → process.executable.path
	procRoots []string          // /proc/<pid>/root prefixes, one per resource with a PID
}

func newBinaryResolver(rps pprofile.ResourceProfilesSlice) *binaryResolver {
	r := &binaryResolver{exeIndex: map[string]string{}}
	for i := 0; i < rps.Len(); i++ {
		attrs := rps.At(i).Resource().Attributes()

		if v, ok := attrs.Get(attrProcessExecutablePath); ok && v.Str() != "" {
			path := v.Str()
			if _, exists := r.exeIndex[filepath.Base(path)]; !exists {
				r.exeIndex[filepath.Base(path)] = path
			}
		}

		if pidStr := pidString(attrs); pidStr != "" {
			r.procRoots = append(r.procRoots, filepath.Join("/proc", pidStr, "root"))
		}
	}
	return r
}

func (r *binaryResolver) empty() bool {
	return len(r.exeIndex) == 0 && len(r.procRoots) == 0
}

// resolve returns the on-disk path for a mapping filename, or "" if no candidate exists on disk.
func (r *binaryResolver) resolve(fname string) string {
	base := filepath.Base(fname)

	// 1. Authoritative executable path keyed by basename.
	if path, ok := r.exeIndex[base]; ok && fileExists(path) {
		return path
	}

	// 2. Process-root fallback: /proc/<pid>/root/<basename>, for shared libraries and for
	//    workloads where only process.pid is known. Try each resource's PID root.
	for _, root := range r.procRoots {
		cand := filepath.Join(root, base)
		if fileExists(cand) {
			return cand
		}
	}
	return ""
}

// pidString extracts process.pid as a decimal string from resource attributes (int or string),
// returning "" when absent.
func pidString(attrs pcommon.Map) string {
	v, ok := attrs.Get(attrProcessPID)
	if !ok {
		return ""
	}
	switch v.Type() {
	case pcommon.ValueTypeInt:
		return strconv.FormatInt(v.Int(), 10)
	case pcommon.ValueTypeStr:
		return v.Str()
	default:
		return ""
	}
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
