package odigosvmprofileattrsprocessor

import (
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

// propagateServiceNameToSamples sets service.name on every profile sample attribute.
// Grafana Pyroscope derives the query label service_name from sample attributes
// (see pkg/ingester/otlp/convert.go serviceNameFromSample), not resource attributes.
func propagateServiceNameToSamples(dict pprofile.ProfilesDictionary, rp pprofile.ResourceProfiles, serviceName string) {
	if serviceName == "" {
		return
	}

	attrIdx := ensureDictionaryServiceNameAttr(dict, serviceName)
	if attrIdx < 0 {
		return
	}

	scopeProfiles := rp.ScopeProfiles()
	for i := 0; i < scopeProfiles.Len(); i++ {
		profiles := scopeProfiles.At(i).Profiles()
		for j := 0; j < profiles.Len(); j++ {
			profile := profiles.At(j)
			samples := profile.Samples()
			for k := 0; k < samples.Len(); k++ {
				ensureSampleHasAttributeIndex(samples.At(k), attrIdx)
			}
		}
	}
}

func ensureDictionaryServiceNameAttr(dict pprofile.ProfilesDictionary, serviceName string) int32 {
	st := dict.StringTable()
	keyIdx := stringTableIndex(st, attrServiceName)
	if keyIdx < 0 {
		st.Append(attrServiceName)
		keyIdx = int32(st.Len() - 1)
	}

	attrTable := dict.AttributeTable()
	for i := 0; i < attrTable.Len(); i++ {
		kv := attrTable.At(i)
		if kv.KeyStrindex() != keyIdx {
			continue
		}
		if kv.Value().Type() == pcommon.ValueTypeStr && kv.Value().AsString() == serviceName {
			return int32(i)
		}
	}

	kv := attrTable.AppendEmpty()
	kv.SetKeyStrindex(keyIdx)
	kv.Value().SetStr(serviceName)
	return int32(attrTable.Len() - 1)
}

func stringTableIndex(st pcommon.StringSlice, want string) int32 {
	for i := 0; i < st.Len(); i++ {
		if st.At(i) == want {
			return int32(i)
		}
	}
	return -1
}

func ensureSampleHasAttributeIndex(sample pprofile.Sample, attrIdx int32) {
	indices := sample.AttributeIndices()
	for i := 0; i < indices.Len(); i++ {
		if indices.At(i) == attrIdx {
			return
		}
	}
	indices.Append(attrIdx)
}
