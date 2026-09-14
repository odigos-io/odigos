package flamegraph

import (
	"testing"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	"github.com/stretchr/testify/require"
	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
	"google.golang.org/protobuf/proto"
)

// pprofFixture builds google/v1 pprof profiles for tests. Index 0 of the string table is the
// empty string, matching the pprof convention the production string lookups rely on.
type pprofFixture struct {
	profile   *googleProfile.Profile
	stringIdx map[string]int64
}

func newPprofFixture() *pprofFixture {
	return &pprofFixture{
		profile:   &googleProfile.Profile{StringTable: []string{""}},
		stringIdx: map[string]int64{"": 0},
	}
}

func (f *pprofFixture) intern(s string) int64 {
	if idx, ok := f.stringIdx[s]; ok {
		return idx
	}
	idx := int64(len(f.profile.StringTable))
	f.profile.StringTable = append(f.profile.StringTable, s)
	f.stringIdx[s] = idx
	return idx
}

func (f *pprofFixture) withSampleType(sampleType, unit string) *pprofFixture {
	f.profile.SampleType = append(f.profile.SampleType, &googleProfile.ValueType{
		Type: f.intern(sampleType),
		Unit: f.intern(unit),
	})
	return f
}

func (f *pprofFixture) withPeriodType(periodType, unit string) *pprofFixture {
	f.profile.PeriodType = &googleProfile.ValueType{
		Type: f.intern(periodType),
		Unit: f.intern(unit),
	}
	return f
}

func (f *pprofFixture) withFunction(id uint64, name string) *pprofFixture {
	f.profile.Function = append(f.profile.Function, &googleProfile.Function{
		Id:   id,
		Name: f.intern(name),
	})
	return f
}

func (f *pprofFixture) withMapping(id uint64, filename, buildID string) *pprofFixture {
	f.profile.Mapping = append(f.profile.Mapping, &googleProfile.Mapping{
		Id:       id,
		Filename: f.intern(filename),
		BuildId:  f.intern(buildID),
	})
	return f
}

// withLocation adds a location whose Line entries reference the given function ids, innermost
// (most deeply inlined) first, which is the ordering pprof uses.
func (f *pprofFixture) withLocation(id uint64, functionIDs ...uint64) *pprofFixture {
	loc := &googleProfile.Location{Id: id}
	for _, fnID := range functionIDs {
		loc.Line = append(loc.Line, &googleProfile.Line{FunctionId: fnID})
	}
	f.profile.Location = append(f.profile.Location, loc)
	return f
}

// withSample adds a sample whose location ids are leaf-first, as pprof stores them.
func (f *pprofFixture) withSample(value int64, leafFirstLocationIDs ...uint64) *pprofFixture {
	f.profile.Sample = append(f.profile.Sample, &googleProfile.Sample{
		LocationId: leafFirstLocationIDs,
		Value:      []int64{value},
	})
	return f
}

func (f *pprofFixture) build() *googleProfile.Profile {
	return f.profile
}

// singleFrameProfile is the smallest mergeable profile: one sample, one location, one function.
func singleFrameProfile(frameName string, value int64) *googleProfile.Profile {
	return newPprofFixture().
		withPeriodType("cpu", "nanoseconds").
		withSampleType("samples", "count").
		withFunction(1, frameName).
		withLocation(1, 1).
		withSample(value, 1).
		build()
}

// otlpFixture builds OTLP profile requests in the shape agents actually send: symbols live in a
// shared ProfilesDictionary and samples reference a stack by index.
type otlpFixture struct {
	dictionary *otelProfile.ProfilesDictionary
	stringIdx  map[string]int32
	stackIdx   map[string]int32
	functionsB map[string]int32
}

func newOtlpFixture() *otlpFixture {
	f := &otlpFixture{
		dictionary: &otelProfile.ProfilesDictionary{
			StringTable: []string{""},
			// Locations default to mapping index 0, so the table must have an entry to resolve.
			MappingTable: []*otelProfile.Mapping{{}},
		},
		stringIdx:  map[string]int32{"": 0},
		stackIdx:   map[string]int32{},
		functionsB: map[string]int32{},
	}
	return f
}

func (f *otlpFixture) intern(s string) int32 {
	if idx, ok := f.stringIdx[s]; ok {
		return idx
	}
	idx := int32(len(f.dictionary.StringTable))
	f.dictionary.StringTable = append(f.dictionary.StringTable, s)
	f.stringIdx[s] = idx
	return idx
}

func (f *otlpFixture) locationForFrame(frameName string) int32 {
	if idx, ok := f.functionsB[frameName]; ok {
		return idx
	}
	fnIdx := int32(len(f.dictionary.FunctionTable))
	f.dictionary.FunctionTable = append(f.dictionary.FunctionTable, &otelProfile.Function{
		NameStrindex: f.intern(frameName),
	})
	locIdx := int32(len(f.dictionary.LocationTable))
	f.dictionary.LocationTable = append(f.dictionary.LocationTable, &otelProfile.Location{
		Lines: []*otelProfile.Line{{FunctionIndex: fnIdx}},
	})
	f.functionsB[frameName] = locIdx
	return locIdx
}

// withStack registers a stack from root-first frame names and returns its index. OTLP stores
// location indices leaf-first, so the frames are reversed on the way in.
func (f *otlpFixture) withStack(rootFirstFrames ...string) int32 {
	key := ""
	indices := make([]int32, 0, len(rootFirstFrames))
	for i := len(rootFirstFrames) - 1; i >= 0; i-- {
		indices = append(indices, f.locationForFrame(rootFirstFrames[i]))
		key += rootFirstFrames[i] + "\x00"
	}
	if idx, ok := f.stackIdx[key]; ok {
		return idx
	}
	idx := int32(len(f.dictionary.StackTable))
	f.dictionary.StackTable = append(f.dictionary.StackTable, &otelProfile.Stack{LocationIndices: indices})
	f.stackIdx[key] = idx
	return idx
}

// allocProfile uses a non-CPU profile type so Pyroscope's converter passes sample values through
// untouched instead of scaling them by the profile period.
func (f *otlpFixture) allocProfile() *otelProfile.Profile {
	return &otelProfile.Profile{
		SampleType:   &otelProfile.ValueType{TypeStrindex: f.intern("alloc_space"), UnitStrindex: f.intern("bytes")},
		PeriodType:   &otelProfile.ValueType{TypeStrindex: f.intern("space"), UnitStrindex: f.intern("bytes")},
		Period:       1,
		TimeUnixNano: 1_700_000_000_000_000_000,
	}
}

func (f *otlpFixture) request(profiles ...*otelProfile.Profile) *pprofileotlp.ExportProfilesServiceRequest {
	return &pprofileotlp.ExportProfilesServiceRequest{
		Dictionary: f.dictionary,
		ResourceProfiles: []*otelProfile.ResourceProfiles{{
			ScopeProfiles: []*otelProfile.ScopeProfiles{{Profiles: profiles}},
		}},
	}
}

func (f *otlpFixture) chunk(t *testing.T, profiles ...*otelProfile.Profile) []byte {
	t.Helper()
	raw, err := proto.Marshal(f.request(profiles...))
	require.NoError(t, err)
	return raw
}

// allocChunkWithStacks is the common case: one OTLP chunk holding one profile whose samples are
// the given root-first stacks with the given weights.
func allocChunkWithStacks(t *testing.T, stacks map[int64][]string) []byte {
	t.Helper()
	f := newOtlpFixture()
	prof := f.allocProfile()
	for value, frames := range stacks {
		prof.Samples = append(prof.Samples, &otelProfile.Sample{
			StackIndex: f.withStack(frames...),
			Values:     []int64{value},
		})
	}
	return f.chunk(t, prof)
}
