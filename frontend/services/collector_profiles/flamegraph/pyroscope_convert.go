// Pyroscope OTLP conversion: when chunk has non-empty dictionary we use
// github.com/grafana/pyroscope/pkg/ingester/otlp.ConvertOtelToGoogle for symbol
// resolution; otherwise fall back to ParseOTLPChunk (frame_N).
package flamegraph

import (
	"reflect"
	"unsafe"

	"google.golang.org/protobuf/encoding/protojson"

	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"

	"github.com/grafana/pyroscope/pkg/ingester/otlp"
	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
)

// ChunksFromPyroscopeOTLP tries to parse chunk as OTLP ExportProfilesServiceRequest (proto JSON),
// run Pyroscope's ConvertOtelToGoogle for each profile, and return samples with resolved symbols.
// Returns (samples, true) on success; (nil, false) when proto unmarshal fails or dictionary is empty
// (caller should fall back to ParseOTLPChunk).
func ChunksFromPyroscopeOTLP(chunk []byte) ([]Sample, bool) {
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	if err := protojson.Unmarshal(chunk, req); err != nil {
		return nil, false
	}
	if req.Dictionary == nil || len(req.Dictionary.StringTable) == 0 {
		return nil, false
	}
	if req.ResourceProfiles == nil {
		return nil, false
	}
	var out []Sample
	for _, rp := range req.ResourceProfiles {
		if rp.ScopeProfiles == nil {
			continue
		}
		for _, sp := range rp.ScopeProfiles {
			if sp.Profiles == nil {
				continue
			}
			for _, p := range sp.Profiles {
				samples := convertProfileViaPyroscope(p, req.Dictionary)
				out = append(out, samples...)
			}
		}
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

// convertProfileViaPyroscope runs Pyroscope's OTLP→pprof conversion and turns the result into our Sample slice.
// Pyroscope returns map[string]convertedProfile (convertedProfile is unexported), so we use reflection to get .profile.
func convertProfileViaPyroscope(src *otelProfile.Profile, dictionary *otelProfile.ProfilesDictionary) []Sample {
	converted, err := otlp.ConvertOtelToGoogle(src, dictionary)
	if err != nil {
		return nil
	}
	var out []Sample
	for _, cp := range converted {
		p := extractGoogleProfile(cp)
		if p != nil {
			out = append(out, googleProfileToSamples(p)...)
		}
	}
	return out
}

// extractGoogleProfile reads the unexported .profile field from otlp.convertedProfile via reflection.
// cp may be an addressable or non-addressable value (e.g. range copy); only use UnsafeAddr when addressable.
func extractGoogleProfile(cp interface{}) *googleProfile.Profile {
	v := reflect.ValueOf(cp)
	if v.Kind() == reflect.Struct {
		f := v.FieldByName("profile")
		if f.IsValid() {
			var p *googleProfile.Profile
			if f.CanInterface() {
				p, _ = f.Interface().(*googleProfile.Profile)
			} else if f.CanAddr() {
				// unexported field: read pointer via unsafe (only when addressable)
				p = *(**googleProfile.Profile)(unsafe.Pointer(f.Addr().UnsafePointer()))
			}
			return p
		}
	}
	return nil
}

// googleProfileToSamples converts a Google pprof Profile to our Sample format (root-first stack, value).
func googleProfileToSamples(p *googleProfile.Profile) []Sample {
	if p == nil || len(p.Sample) == 0 || len(p.StringTable) == 0 {
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
	getName := func(stringIdx int64) string {
		if stringIdx >= 0 && int(stringIdx) < len(p.StringTable) {
			return p.StringTable[stringIdx]
		}
		return ""
	}
	var out []Sample
	for _, s := range p.Sample {
		var value int64
		for _, v := range s.Value {
			value += v
		}
		if value <= 0 {
			value = 1
		}
		stack := make([]string, 0, len(s.LocationId))
		for i := len(s.LocationId) - 1; i >= 0; i-- {
			locID := s.LocationId[i]
			loc := locByID[locID]
			if loc == nil || len(loc.Line) == 0 {
				continue
			}
			line := loc.Line[0]
			fn := funcByID[line.FunctionId]
			if fn == nil {
				continue
			}
			name := getName(fn.Name)
			if name != "" {
				stack = append(stack, name)
			}
		}
		if len(stack) > 0 {
			out = append(out, Sample{Stack: stack, Value: value})
		}
	}
	return out
}
