// Pyroscope OTLP conversion: github.com/grafana/pyroscope/pkg/ingester/otlp.ConvertOtelToGoogle
// (same code path as Grafana Pyroscope ingest). Used only via SamplesFromOTLPChunk.
package flamegraph

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"google.golang.org/protobuf/encoding/protojson"

	pprofileotlp "go.opentelemetry.io/proto/otlp/collector/profiles/v1development"
	otelCommon "go.opentelemetry.io/proto/otlp/common/v1"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
	otelResource "go.opentelemetry.io/proto/otlp/resource/v1"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	"github.com/grafana/pyroscope/pkg/ingester/otlp"
)

var protoJSONUnmarshal = protojson.UnmarshalOptions{DiscardUnknown: true}
const pyroscopeServiceNameKey = "service.name"

// tryPyroscopeOTLP parses OTLP profile JSON as ExportProfilesServiceRequest and converts via
// Pyroscope's ConvertOtelToGoogle. Returns ok=false with a short reason when this path cannot be used.
func tryPyroscopeOTLP(chunk []byte) (samples []Sample, ok bool, failReason string) {
	req := &pprofileotlp.ExportProfilesServiceRequest{}
	if err := protoJSONUnmarshal.Unmarshal(chunk, req); err != nil {
		return nil, false, fmt.Sprintf("protojson_unmarshal: %v", err)
	}
	if req.Dictionary == nil || len(req.Dictionary.StringTable) == 0 {
		return nil, false, "missing_or_empty_dictionary_string_table"
	}
	if len(req.ResourceProfiles) == 0 {
		return nil, false, "no_resource_profiles"
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
				normalizeSampleValuesForPyroscope(p)
				ensureServiceNameInSamples(p, req.Dictionary, rp.Resource)
				samples, extracted, extractReason := convertProfileViaPyroscope(p, req.Dictionary)
				if !extracted {
					if extractReason == "" {
						extractReason = "unknown"
					}
					return nil, false, "pyroscope_profile_extract_failed:" + extractReason
				}
				out = append(out, samples...)
			}
		}
	}
	if len(out) == 0 {
		return nil, false, "convert_otel_to_google_yielded_no_samples"
	}
	return out, true, ""
}

// normalizeSampleValuesForPyroscope adapts chunks where sample values carry multiple counters
// but sample type contains a single entry. Pyroscope conversion expects lengths to match.
func normalizeSampleValuesForPyroscope(profile *otelProfile.Profile) {
	if profile == nil {
		return
	}
	for _, s := range profile.Samples {
		if s == nil || len(s.Values) <= 1 {
			continue
		}
		sum := int64(0)
		for _, v := range s.Values {
			sum += v
		}
		s.Values = []int64{sum}
	}
}

func ensureServiceNameInSamples(profile *otelProfile.Profile, dictionary *otelProfile.ProfilesDictionary, resource *otelResource.Resource) {
	if profile == nil || dictionary == nil {
		return
	}

	serviceName := "odigos-profile"
	if resource != nil {
		if svc := getResourceAttributeString(resource, pyroscopeServiceNameKey); svc != "" {
			serviceName = svc
		}
	}

	keyIdx := ensureStringInDictionary(dictionary, pyroscopeServiceNameKey)
	_ = ensureStringInDictionary(dictionary, serviceName)

	attrIdx := int32(len(dictionary.AttributeTable))
	dictionary.AttributeTable = append(dictionary.AttributeTable, &otelProfile.KeyValueAndUnit{
		KeyStrindex: keyIdx,
		Value: &otelCommon.AnyValue{
			Value: &otelCommon.AnyValue_StringValue{StringValue: serviceName},
		},
	})

	for _, s := range profile.Samples {
		if s == nil {
			continue
		}
		if svc := getDictionaryAttributeString(s.AttributeIndices, dictionary, pyroscopeServiceNameKey); svc != "" {
			continue
		}
		s.AttributeIndices = append(s.AttributeIndices, attrIdx)
	}
}

func ensureStringInDictionary(dictionary *otelProfile.ProfilesDictionary, value string) int32 {
	for i, v := range dictionary.StringTable {
		if v == value {
			return int32(i)
		}
	}
	dictionary.StringTable = append(dictionary.StringTable, value)
	return int32(len(dictionary.StringTable) - 1)
}

func getDictionaryAttributeString(attributeIndices []int32, dictionary *otelProfile.ProfilesDictionary, key string) string {
	for _, idx := range attributeIndices {
		if idx < 0 || int(idx) >= len(dictionary.AttributeTable) {
			continue
		}
		kv := dictionary.AttributeTable[idx]
		if kv == nil {
			continue
		}
		keyIdx := kv.KeyStrindex
		if keyIdx < 0 || int(keyIdx) >= len(dictionary.StringTable) {
			continue
		}
		if dictionary.StringTable[keyIdx] != key {
			continue
		}
		if kv.Value != nil {
			return kv.Value.GetStringValue()
		}
	}
	return ""
}

func getResourceAttributeString(resource *otelResource.Resource, key string) string {
	if resource == nil {
		return ""
	}
	for _, kv := range resource.Attributes {
		if kv == nil {
			continue
		}
		if kv.Key == key && kv.Value != nil {
			return kv.Value.GetStringValue()
		}
	}
	return ""
}

// convertProfileViaPyroscope runs Pyroscope's OTLP→pprof conversion and turns the result into our Sample slice.
func convertProfileViaPyroscope(src *otelProfile.Profile, dictionary *otelProfile.ProfilesDictionary) ([]Sample, bool, string) {
	converted, err := otlp.ConvertOtelToGoogle(src, dictionary)
	if err != nil {
		return nil, false, "convert_error:" + err.Error()
	}
	var out []Sample
	extractedAny := false
	for _, cp := range converted {
		cp := cp // local copy so &cp is addressable for unexported field reflection
		p := extractGoogleProfile(&cp)
		if p != nil {
			extractedAny = true
			out = append(out, googleProfileToSamples(p)...)
		}
	}
	if extractedAny {
		return out, true, ""
	}
	return nil, false, fmt.Sprintf("converted_profiles=%d", len(converted))
}

// extractGoogleProfile reads converted profile data from pyroscope conversion output.
// Newer pyroscope versions may change field names, so we locate by type first.
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
