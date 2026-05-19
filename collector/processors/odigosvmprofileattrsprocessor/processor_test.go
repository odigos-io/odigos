package odigosvmprofileattrsprocessor

import (
	"testing"

	"github.com/odigos-io/odigos/common/unixfd"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"
)

func TestProfileAttrCache_RegisterUnregister(t *testing.T) {
	cache := newProfileAttrCache()
	cache.applyEvent(unixfd.EncodeLogsAttrRegister(42, "service.name:payment-api,odigos.vm.source.kind:systemd"))
	packed, ok := cache.get(42)
	require.True(t, ok)
	require.Equal(t, "service.name:payment-api,odigos.vm.source.kind:systemd", packed)

	cache.applyEvent(unixfd.EncodeLogsAttrUnregister(42))
	_, ok = cache.get(42)
	require.False(t, ok)
}

func TestApplyPackedResourceAttributes(t *testing.T) {
	attrs := pcommon.NewMap()
	err := applyPackedResourceAttributes(attrs, "service.name:my-svc,odigos.vm.source.kind:docker")
	require.NoError(t, err)
	require.Equal(t, "my-svc", attrs.AsRaw()["service.name"])
	require.Equal(t, "docker", attrs.AsRaw()["odigos.vm.source.kind"])
}

func TestProcessProfiles_DropsUnregisteredPID(t *testing.T) {
	proc := &vmProfileAttrsProcessor{
		logger:    zap.NewNop(),
		cfg:       &Config{},
		attrCache: newProfileAttrCache(),
	}

	profiles := pprofile.NewProfiles()
	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutInt(attrProcessPID, 99)

	out, err := proc.processProfiles(t.Context(), profiles)
	require.NoError(t, err)
	require.False(t, profilesExportable(out))
}

func TestProcessProfiles_EnrichesRegisteredPID(t *testing.T) {
	proc := &vmProfileAttrsProcessor{
		logger:    zap.NewNop(),
		cfg:       &Config{},
		attrCache: newProfileAttrCache(),
	}
	proc.attrCache.applyEvent(unixfd.EncodeLogsAttrRegister(10, "service.name:orders-api"))

	profiles := pprofile.NewProfiles()
	profiles.Dictionary().StringTable().Append(attrServiceName)
	rp := profiles.ResourceProfiles().AppendEmpty()
	rp.Resource().Attributes().PutInt(attrProcessPID, 10)
	rp.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty().Samples().AppendEmpty()

	out, err := proc.processProfiles(t.Context(), profiles)
	require.NoError(t, err)
	require.Equal(t, 1, out.ResourceProfiles().Len())
	svc, ok := out.ResourceProfiles().At(0).Resource().Attributes().Get(attrServiceName)
	require.True(t, ok)
	require.Equal(t, "orders-api", svc.AsString())

	sample := out.ResourceProfiles().At(0).ScopeProfiles().At(0).Profiles().At(0).Samples().At(0)
	require.Equal(t, 1, sample.AttributeIndices().Len())
	attr := out.Dictionary().AttributeTable().At(int(sample.AttributeIndices().At(0)))
	require.Equal(t, "orders-api", attr.Value().AsString())
}

func TestProcessProfiles_PartialBatchKeepsRegisteredOnly(t *testing.T) {
	proc := &vmProfileAttrsProcessor{
		logger:    zap.NewNop(),
		cfg:       &Config{},
		attrCache: newProfileAttrCache(),
	}
	proc.attrCache.applyEvent(unixfd.EncodeLogsAttrRegister(10, "service.name:frontend"))
	proc.attrCache.applyEvent(unixfd.EncodeLogsAttrRegister(30, "service.name:coupon"))

	profiles := pprofile.NewProfiles()
	for _, pid := range []int64{10, 20, 30} {
		rp := profiles.ResourceProfiles().AppendEmpty()
		rp.Resource().Attributes().PutInt(attrProcessPID, pid)
	}

	out, err := proc.processProfiles(t.Context(), profiles)
	require.NoError(t, err)
	require.Equal(t, 2, out.ResourceProfiles().Len())

	names := make([]string, 0, 2)
	for i := 0; i < out.ResourceProfiles().Len(); i++ {
		svc, ok := out.ResourceProfiles().At(i).Resource().Attributes().Get(attrServiceName)
		require.True(t, ok)
		names = append(names, svc.AsString())
	}
	require.ElementsMatch(t, []string{"frontend", "coupon"}, names)
}

func TestProcessProfiles_DropsResourceWithoutPID(t *testing.T) {
	proc := &vmProfileAttrsProcessor{
		logger:    zap.NewNop(),
		cfg:       &Config{},
		attrCache: newProfileAttrCache(),
	}
	proc.attrCache.applyEvent(unixfd.EncodeLogsAttrRegister(10, "service.name:orders-api"))

	profiles := pprofile.NewProfiles()
	withPID := profiles.ResourceProfiles().AppendEmpty()
	withPID.Resource().Attributes().PutInt(attrProcessPID, 10)
	withoutPID := profiles.ResourceProfiles().AppendEmpty()
	withoutPID.Resource().Attributes().PutStr("service.name", "should-not-export")

	out, err := proc.processProfiles(t.Context(), profiles)
	require.NoError(t, err)
	require.Equal(t, 1, out.ResourceProfiles().Len())
}

func TestProcessProfiles_EmptyInput(t *testing.T) {
	proc := &vmProfileAttrsProcessor{
		logger:    zap.NewNop(),
		cfg:       &Config{},
		attrCache: newProfileAttrCache(),
	}

	out, err := proc.processProfiles(t.Context(), pprofile.NewProfiles())
	require.NoError(t, err)
	require.False(t, profilesExportable(out))
}
