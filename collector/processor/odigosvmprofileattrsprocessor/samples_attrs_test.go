package odigosvmprofileattrsprocessor

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func TestPropagateServiceNameToSamples(t *testing.T) {
	profiles := pprofile.NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append(attrServiceName)

	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutStr(attrServiceName, "odigos-demo-frontend")

	sp := rp.ScopeProfiles().AppendEmpty()
	prof := sp.Profiles().AppendEmpty()
	prof.Samples().AppendEmpty()

	propagateServiceNameToSamples(dict, rp, "odigos-demo-frontend", map[string]int32{})

	sample := prof.Samples().At(0)
	require.Equal(t, 1, sample.AttributeIndices().Len())
	attr := dict.AttributeTable().At(int(sample.AttributeIndices().At(0)))
	require.Equal(t, attrServiceName, dict.StringTable().At(int(attr.KeyStrindex())))
	require.Equal(t, "odigos-demo-frontend", attr.Value().AsString())
}

// TestPropagateServiceNameToSamples_CacheReuse verifies that two resource profiles sharing a
// service name reuse a single dictionary attribute-table entry via the per-batch cache, rather
// than appending a duplicate entry.
func TestPropagateServiceNameToSamples_CacheReuse(t *testing.T) {
	profiles := pprofile.NewProfiles()
	dict := profiles.Dictionary()
	dict.StringTable().Append(attrServiceName)

	cache := map[string]int32{}
	baseAttrs := dict.AttributeTable().Len()

	for pid := 0; pid < 2; pid++ {
		rp := profiles.ResourceProfiles().AppendEmpty()
		sp := rp.ScopeProfiles().AppendEmpty()
		prof := sp.Profiles().AppendEmpty()
		prof.Samples().AppendEmpty()

		propagateServiceNameToSamples(dict, rp, "orders-api", cache)

		sample := prof.Samples().At(0)
		require.Equal(t, 1, sample.AttributeIndices().Len())
		require.Equal(t, cache["orders-api"], sample.AttributeIndices().At(0))
	}

	// Only one new attribute-table entry for the shared service name across both resources.
	require.Equal(t, baseAttrs+1, dict.AttributeTable().Len())
	require.Len(t, cache, 1)
}

func TestStringTableIndex(t *testing.T) {
	st := pcommon.NewStringSlice()
	st.Append("foo", attrServiceName)
	require.Equal(t, int32(1), stringTableIndex(st, attrServiceName))
	require.Equal(t, int32(-1), stringTableIndex(st, "missing"))
}
