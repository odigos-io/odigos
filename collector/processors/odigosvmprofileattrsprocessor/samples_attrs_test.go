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

	propagateServiceNameToSamples(dict, rp, "odigos-demo-frontend")

	sample := prof.Samples().At(0)
	require.Equal(t, 1, sample.AttributeIndices().Len())
	attr := dict.AttributeTable().At(int(sample.AttributeIndices().At(0)))
	require.Equal(t, attrServiceName, dict.StringTable().At(int(attr.KeyStrindex())))
	require.Equal(t, "odigos-demo-frontend", attr.Value().AsString())
}

func TestStringTableIndex(t *testing.T) {
	st := pcommon.NewStringSlice()
	st.Append("foo", attrServiceName)
	require.Equal(t, int32(1), stringTableIndex(st, attrServiceName))
	require.Equal(t, int32(-1), stringTableIndex(st, "missing"))
}
