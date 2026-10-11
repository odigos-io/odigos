// Package flamegraph converts stored OTLP Profiles JSON chunks into merged stack samples and Pyroscope-shaped flame JSON.
//
// Pyroscope OTLP conversion uses github.com/grafana/pyroscope/pkg/ingester/otlp.ConvertOtelToGoogle
// (same path as Grafana Pyroscope ingest). Public Pyroscope modules expose conversion on protobuf
// types; chunks may be OTLP JSON or binary protobuf (pdata ProtoMarshaler, same wire as ExportProfilesServiceRequest body).
//
// Pipeline in this file:
//  1. ParseExportProfilesServiceRequest / tryPyroscopeOTLP: binary proto (OdigosProfilesConsumer) or JSON → ExportProfilesServiceRequest.
//  2. googleProfilesFromParsedRequest → Google pprof per OTLP profile (Pyroscope ConvertOtelToGoogle).
//  3. mergeGoogleProfilesGrouped (pkg/pprof.ProfileMerge) across profiles per compatibility bucket; googleProfileToSamples
//     is used only for legacy/sample-based helpers (multi-Line Locations, placeholders for missing location ids).
//  4. profile_builder uses BuildFlamebearerViaPyroscopeSymdb: merged Google profile → pkg/pprof.Normalize → symdb →
//     Resolver.Tree → pkg/model NewFlameGraph + ExportToFlamebearer (same as Pyroscope Explore stack merge path).
package flamegraph

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"unsafe"

	"google.golang.org/protobuf/proto"

	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	typesv1 "github.com/grafana/pyroscope/api/gen/proto/go/types/v1"
)

// ParseExportProfilesServiceRequest decodes one OTLP Profiles chunk.
func ParseExportProfilesServiceRequest(chunk []byte) (*pprofileotlp.ExportProfilesServiceRequest, error) {
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	err := proto.Unmarshal(chunk, req)
	if err == nil && len(req.ResourceProfiles) > 0 {
		return req, nil
	}
	if len(req.ResourceProfiles) == 0 {
		return nil, fmt.Errorf("no resource profiles found during proto unmarshal")
	}
	return req, nil
}

// DefaultProfileType is used when OTLP chunks do not expose a usable pprof sample type.
func DefaultProfileType() *typesv1.ProfileType {
	return &typesv1.ProfileType{SampleType: "cpu"}
}

func profileTypeFromGoogleProfile(p *googleProfile.Profile) *typesv1.ProfileType {
	if p == nil || len(p.SampleType) == 0 {
		return DefaultProfileType()
	}
	idx := int(p.DefaultSampleType)
	if idx < 0 || idx >= len(p.SampleType) {
		idx = 0
	}
	st := p.SampleType[idx]
	if st == nil {
		return DefaultProfileType()
	}
	sampleType := stringFromPprofStringTable(p.StringTable, st.Type)
	sampleUnit := stringFromPprofStringTable(p.StringTable, st.Unit)
	if sampleType == "" {
		return DefaultProfileType()
	}
	return &typesv1.ProfileType{
		SampleType: sampleType,
		SampleUnit: sampleUnit,
	}
}

func stringFromPprofStringTable(table []string, idx int64) string {
	if idx <= 0 || int(idx) >= len(table) {
		return ""
	}
	return table[idx]
}

// normalizeSampleValuesForPyroscope fixes OTLP Profile samples so Pyroscope's OTLP→pprof conversion
// sees a shape it accepts:
//
//   - Some agents (including certain eBPF profiler paths) send TimestampsUnixNano but leave Values empty.
//     ConvertOtelToGoogle expects at least one numeric Value per sample. We synthesize a single value:
//     the number of timestamps in that sample (one aggregate weight, not one Value per timestamp).
//     That preserves a non-zero weight so the sample is not dropped while staying scalar.
//   - If multiple Values are present (multi-dimensional sample types) but downstream expects one column,
//     we keep only the first Value to avoid mixing incompatible units in the flame merge path.
func normalizeSampleValuesForPyroscope(profile *otelProfile.Profile) {
	if profile == nil {
		return
	}
	for _, s := range profile.Samples {
		if s == nil {
			continue
		}
		if len(s.Values) == 0 && len(s.TimestampsUnixNano) > 0 {
			s.Values = []int64{int64(len(s.TimestampsUnixNano))}
			continue
		}
		if len(s.Values) <= 1 {
			continue
		}
		s.Values = []int64{s.Values[0]}
	}
}

// extractGoogleProfile retrieves the *googleProfile.Profile from a Pyroscope ConvertedProfile value.
//
// otlp.ConvertOtelToGoogle returns []ConvertedProfile where ConvertedProfile is an unexported struct
// in github.com/grafana/pyroscope/pkg/ingester/otlp. The Profile field we need is also unexported in
// that struct, so we cannot access it through the normal Go type system. We resolve it by:
//  1. Scanning exported fields by type (works if Pyroscope ever makes the field exported).
//  2. Using reflect + unsafe to read unexported pointer/value fields by memory layout (current approach).
//  3. Falling back to JSON round-trip via the "profile" key if both fail.
//
// Keep the grafana/pyroscope dependency pinned so the struct layout stays stable across builds.
func extractGoogleProfile(cp interface{}) *googleProfile.Profile {
	v := reflect.ValueOf(cp)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct {
		return nil
	}

	googleProfilePtrType := reflect.TypeOf((*googleProfile.Profile)(nil))
	googleProfileValType := reflect.TypeOf(googleProfile.Profile{})

	for i := 0; i < v.NumField(); i++ {
		f := v.Field(i)
		t := f.Type()

		switch t {
		case googleProfilePtrType:
			if f.CanInterface() {
				if p, _ := f.Interface().(*googleProfile.Profile); p != nil {
					return p
				}
			}
			// Pyroscope's converted type may use unexported fields; unsafe dereference couples us to their
			// layout — keep collector deps pinned and rely on tests + JSON fallback below.
			if f.CanAddr() {
				if p := *(**googleProfile.Profile)(unsafe.Pointer(f.Addr().UnsafePointer())); p != nil {
					return p
				}
			}
		case googleProfileValType:
			if f.CanInterface() {
				if p, _ := f.Interface().(googleProfile.Profile); p.Sample != nil {
					cp := p
					return &cp
				}
			}
		}
	}

	// Fallback: if converted profile exposes JSON fields, decode "profile".
	raw, err := json.Marshal(cp)
	if err != nil {
		return nil
	}
	var holder struct {
		Profile *googleProfile.Profile `json:"profile"`
	}
	if err := json.Unmarshal(raw, &holder); err != nil {
		return nil
	}
	return holder.Profile
}

// maxFrameNameLen caps display strings for UI / merge stability (aligned with common pprof truncation).
const maxFrameNameLen = 256

func truncateFrameName(s string) string {
	const maxRunes = maxFrameNameLen
	var b strings.Builder
	n := 0
	for _, r := range s {
		if n >= maxRunes {
			b.WriteString("…")
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

func stringFromTable(table []string, idx int64) string {
	if idx >= 0 && int(idx) < len(table) {
		return table[idx]
	}
	return ""
}

func functionLineLabel(p *googleProfile.Profile, fn *googleProfile.Function, sourceLine int64) string {
	if fn == nil {
		return ""
	}
	if s := stringFromTable(p.StringTable, fn.Name); s != "" {
		return truncateFrameName(s)
	}
	if s := stringFromTable(p.StringTable, fn.SystemName); s != "" {
		return truncateFrameName(s)
	}
	file := stringFromTable(p.StringTable, fn.Filename)
	if file != "" && sourceLine > 0 {
		return truncateFrameName(fmt.Sprintf("%s:%d", file, sourceLine))
	}
	if file != "" && fn.StartLine > 0 {
		return truncateFrameName(fmt.Sprintf("%s:%d", file, fn.StartLine))
	}
	if file != "" {
		return truncateFrameName(file)
	}
	return ""
}

func lineFrameLabel(p *googleProfile.Profile, ln *googleProfile.Line, funcByID map[uint64]*googleProfile.Function) string {
	if ln == nil {
		return ""
	}
	return functionLineLabel(p, funcByID[ln.FunctionId], ln.Line)
}

// locationFallbackLabel is used when a Location has no usable Line entries (same as classic pprof mapping+PC).
func locationFallbackLabel(p *googleProfile.Profile, loc *googleProfile.Location, mappingByID map[uint64]*googleProfile.Mapping) string {
	if loc == nil {
		return truncateFrameName("[unknown frame]")
	}
	if m := mappingByID[loc.MappingId]; m != nil {
		fnStr := stringFromTable(p.StringTable, m.Filename)
		buildID := stringFromTable(p.StringTable, m.BuildId)
		if fnStr != "" && loc.Address != 0 {
			if buildID != "" {
				return truncateFrameName(fmt.Sprintf("%s [%s]+0x%x", fnStr, buildID, loc.Address))
			}
			return truncateFrameName(fmt.Sprintf("%s+0x%x", fnStr, loc.Address))
		}
		if fnStr != "" {
			return truncateFrameName(fnStr)
		}
		if buildID != "" && loc.Address != 0 {
			return truncateFrameName(fmt.Sprintf("%s+0x%x", buildID, loc.Address))
		}
	}
	if loc.Address != 0 {
		return fmt.Sprintf("0x%x", loc.Address)
	}
	return fmt.Sprintf("frame_%d", loc.Id)
}

// locationFrameLabels returns one or more flame labels for a pprof Location.
// For inlined symbols, google/v1 profile puts multiple Line entries on one Location; the last line is
// the caller into which preceding lines were inlined. We emit outer→inner (caller first) so that,
// when building a root-first stack (root … leaf), inline chains stay in natural call order — closer to
// Pyroscope/Grafana than collapsing the whole PC to a single picked line.
func locationFrameLabels(p *googleProfile.Profile, loc *googleProfile.Location, funcByID map[uint64]*googleProfile.Function, mappingByID map[uint64]*googleProfile.Mapping) []string {
	if loc == nil {
		return []string{truncateFrameName("[unknown frame]")}
	}
	if len(loc.Line) > 0 {
		labels := make([]string, 0, len(loc.Line))
		for i := len(loc.Line) - 1; i >= 0; i-- {
			if s := lineFrameLabel(p, loc.Line[i], funcByID); s != "" {
				labels = append(labels, s)
			}
		}
		if len(labels) > 0 {
			return labels
		}
	}
	return []string{locationFallbackLabel(p, loc, mappingByID)}
}

// googleProfileToSamples converts a Google pprof Profile to our Sample format (root-first stack, value).
func googleProfileToSamples(p *googleProfile.Profile) []Sample {
	if p == nil || len(p.Sample) == 0 {
		return nil
	}
	locByID := make(map[uint64]*googleProfile.Location)
	for _, loc := range p.Location {
		locByID[loc.Id] = loc
	}
	funcByID := make(map[uint64]*googleProfile.Function)
	for _, fn := range p.Function {
		funcByID[fn.Id] = fn
	}
	mappingByID := make(map[uint64]*googleProfile.Mapping)
	for _, m := range p.Mapping {
		if m != nil {
			mappingByID[m.Id] = m
		}
	}
	var out []Sample
	for _, s := range p.Sample {
		var value int64
		if len(s.Value) > 0 {
			// Multiple values map to different sample types; summing mixes units (e.g. count + ns).
			value = s.Value[0]
		}
		if value <= 0 {
			continue
		}
		// Keep samples even when a location id is missing after merge: dropping them creates visible
		// holes vs Pyroscope/Grafana, which retain weight by resolving through their symbol pipeline.
		stack := make([]string, 0, len(s.LocationId)*2)
		for i := len(s.LocationId) - 1; i >= 0; i-- {
			locID := s.LocationId[i]
			loc := locByID[locID]
			if loc == nil {
				stack = append(stack, truncateFrameName(fmt.Sprintf("[missing location id=%d]", locID)))
				continue
			}
			stack = append(stack, locationFrameLabels(p, loc, funcByID, mappingByID)...)
		}
		if len(stack) > 0 {
			out = append(out, Sample{Stack: stack, Value: value})
		}
	}
	return out
}
