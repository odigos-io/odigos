package odigosvmprofileattrsprocessor

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pprofile"
	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/unixfd"
)

// resetSharedForTest cancels the singleton's client goroutine and resets package state
// so each test starts clean and goleak (the generated TestMain) sees no lingering
// goroutine. Production NEVER calls this — the client lives for the process lifetime.
func resetSharedForTest(t *testing.T) {
	t.Helper()
	t.Cleanup(func() {
		if sharedCancel != nil {
			sharedCancel()
			<-sharedDone
		}
		sharedOnce = sync.Once{}
		sharedCache = nil
		sharedCancel = nil
		sharedDone = nil
	})
}

func TestProfileAttrCache_RegisterUnregister(t *testing.T) {
	cache := newProfileAttrCache()
	cache.applyEvent(unixfd.EncodeAttrRegister(42, "service.name:payment-api,odigos.vm.source.kind:systemd"))
	packed, ok := cache.get(42)
	require.True(t, ok)
	require.Equal(t, "service.name:payment-api,odigos.vm.source.kind:systemd", packed)

	cache.applyEvent(unixfd.EncodeAttrUnregister(42))
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
	proc.attrCache.applyEvent(unixfd.EncodeAttrRegister(10, "service.name:orders-api"))

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
	proc.attrCache.applyEvent(unixfd.EncodeAttrRegister(10, "service.name:frontend"))
	proc.attrCache.applyEvent(unixfd.EncodeAttrRegister(30, "service.name:coupon"))

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
	proc.attrCache.applyEvent(unixfd.EncodeAttrRegister(10, "service.name:orders-api"))

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

// TestProfileAttrCache_ResetWipesState confirms reset() wipes the cache so a fresh snapshot
// rebuild on reconnect doesn't have stale entries left over from the previous session.
func TestProfileAttrCache_ResetWipesState(t *testing.T) {
	c := newProfileAttrCache()
	c.applyEvent(unixfd.EncodeAttrRegister(1, "service.name:a"))
	c.applyEvent(unixfd.EncodeAttrRegister(2, "service.name:b"))
	require.Equal(t, 2, c.size())

	c.reset()
	require.Equal(t, 0, c.size())
	_, ok := c.get(1)
	require.False(t, ok)
}

// TestSharedCache_SurvivesProcessorRebuild verifies the fix for the SIGHUP cache-loss
// bug: the PID→attrs cache is process-global, so a config reload (which destroys and
// rebuilds the processor) leaves the cache warm. Two start() calls — simulating the
// pre- and post-reload processor instances — must share the same cache, and an entry
// registered before the "reload" must survive it.
func TestSharedCache_SurvivesProcessorRebuild(t *testing.T) {
	resetSharedForTest(t)
	cfg := &Config{SocketPath: "/tmp/odigos-test-nonexistent.sock"} // client retries harmlessly

	p1 := &vmProfileAttrsProcessor{logger: zap.NewNop(), cfg: cfg}
	require.NoError(t, p1.start(t.Context(), nil))

	// Register a PID via the shared cache (as the live unixfd stream would).
	p1.attrCache.applyEvent(unixfd.EncodeAttrRegister(7, "service.name:catalog"))
	packed, ok := p1.attrCache.get(7)
	require.True(t, ok)
	require.Equal(t, "service.name:catalog", packed)

	// Simulate a config reload: the old processor is shut down and a new one is built.
	require.NoError(t, p1.shutdown(t.Context()))
	p2 := &vmProfileAttrsProcessor{logger: zap.NewNop(), cfg: cfg}
	require.NoError(t, p2.start(t.Context(), nil))

	// The rebuilt processor must reference the SAME cache (not a fresh empty one)...
	require.Same(t, p1.attrCache, p2.attrCache, "cache must survive the processor rebuild")
	// ...and the entry registered before the reload must still be there.
	packed, ok = p2.attrCache.get(7)
	require.True(t, ok, "entry must survive the reload (no cache wipe)")
	require.Equal(t, "service.name:catalog", packed)
}
