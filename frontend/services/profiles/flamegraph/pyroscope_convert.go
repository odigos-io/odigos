// Package flamegraph converts stored OTLP profile chunks (protobuf ExportProfilesServiceRequest wire)
// into merged stack samples and Pyroscope-shaped flame JSON.
//
// Pyroscope OTLP conversion uses github.com/grafana/pyroscope/pkg/ingester/otlp.ConvertOtelToGoogle
// (same path as Grafana Pyroscope ingest).
//
// Pipeline in this file:
//  1. tryPyroscopeOTLP: proto.Unmarshal each chunk → ExportProfilesServiceRequest.
//  2. Per profile: normalizeSampleValuesForPyroscope (value length vs sample type), ensureServiceNameInSamples
//     (Pyroscope expects service.name on samples), then convertProfileViaPyroscope → Google pprof-style frames.
//  3. convertProfileViaPyroscope wraps otlp.ConvertOtelToGoogle and maps LocationLines to our Sample{Stack, Value}.
//  4. SamplesFromOTLPChunk is the entry used by profile_builder.
package flamegraph

import (
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/odigos-io/odigos/frontend/services/profiles/otlpchunk"

	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	otelCommon "go.opentelemetry.io/proto/otlp/common/v1"
	otelProfile "go.opentelemetry.io/proto/otlp/profiles/v1development"
	otelResource "go.opentelemetry.io/proto/otlp/resource/v1"

	googleProfile "github.com/grafana/pyroscope/api/gen/proto/go/google/v1"
	"github.com/grafana/pyroscope/pkg/ingester/otlp"
)

const (
	// failReasonProfileExtractPrefix is prefixed to ConvertOtelToGoogle extract failures for logging.
	failReasonProfileExtractPrefix = "pyroscope_profile_extract_failed:"
	failReasonNoSamplesFromConvert = "convert_otel_to_google_yielded_no_samples"
)

// tryPyroscopeOTLP unmarshals one OTLP protobuf chunk, runs Pyroscope's ConvertOtelToGoogle per
// profile, and maps the result to Sample slices. ok is false when the payload is invalid or conversion
// produces no extractable stacks (failReason is for diagnostics/logging).
func tryPyroscopeOTLP(chunk []byte) (samples []Sample, ok bool, failReason string) {
	if len(chunk) == 0 {
		return nil, false, "empty_chunk"
	}
	req, err := otlpchunk.UnmarshalExportProfilesRequest(chunk)
	if err != nil {
		return nil, false, fmt.Sprintf("proto_unmarshal: %v", err)
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
					return nil, false, failReasonProfileExtractPrefix + extractReason
				}
				out = append(out, samples...)
			}
		}
	}
	if len(out) == 0 {
		return nil, false, failReasonNoSamplesFromConvert
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

// ensureServiceNameInSamples adds service.name to profile samples when missing, so Pyroscope's
// converter matches ingest expectations. defaultProfileServiceName is used only when the OTLP
// resource has no service.name (e.g. some agent paths); prefer the real k8s workload/service attribute.
func ensureServiceNameInSamples(profile *otelProfile.Profile, dictionary *otelProfile.ProfilesDictionary, resource *otelResource.Resource) {
	if profile == nil || dictionary == nil {
		return
	}

	const defaultProfileServiceName = "odigos-profile"
	serviceName := defaultProfileServiceName

	if resource != nil {
		if svc := getResourceAttributeString(resource, string(semconv.ServiceNameKey)); svc != "" {
			serviceName = svc
		}
	}

	keyIdx := ensureStringInDictionary(dictionary, string(semconv.ServiceNameKey))
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
		if svc := getDictionaryAttributeString(s.AttributeIndices, dictionary, string(semconv.ServiceNameKey)); svc != "" {
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
		// LocationId follows pprof / Pyroscope OTLP conversion order: outer frame first (matches stack.LocationIndices).
		for i := 0; i < len(s.LocationId); i++ {
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
