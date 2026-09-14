package profiles

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSourceIDFromStringsNormalizesTheKind(t *testing.T) {
	id, err := SourceIDFromStrings(profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	assert.Equal(t, profilesTestNamespace, id.Namespace)
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, id.Kind)
	assert.Equal(t, profilesTestWorkload, id.Name)
}

func TestSourceIDFromStringsRequiresEveryPartOfTheIdentity(t *testing.T) {
	cases := map[string][3]string{
		"no namespace": {"", "Deployment", profilesTestWorkload},
		"no kind":      {profilesTestNamespace, "", profilesTestWorkload},
		"no name":      {profilesTestNamespace, "Deployment", ""},
		"nothing":      {"", "", ""},
	}

	for name, args := range cases {
		t.Run(name, func(t *testing.T) {
			id, err := SourceIDFromStrings(args[0], args[1], args[2])

			require.Error(t, err)
			assert.Contains(t, err.Error(), "missing namespace, kind, or name")
			assert.Empty(t, id)
		})
	}
}

func TestEnableProfilingForSourceOpensASlot(t *testing.T) {
	store := newProfilesTestStore(t)

	out, err := EnableProfilingForSource(store, profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	assert.Equal(t, "ok", out.Status)
	assert.Equal(t, "checkout-ns/Deployment/checkout-service", out.SourceKey)
	assert.Equal(t, 4, out.MaxSlots)
	assert.Equal(t, 1, out.ActiveSlots)
	assert.True(t, store.IsActive("checkout-ns/Deployment/checkout-service"))
}

func TestEnableProfilingForSourceCountsEveryOpenSlot(t *testing.T) {
	store := newProfilesTestStore(t)
	_, err := EnableProfilingForSource(store, profilesTestNamespace, "deployment", "first")
	require.NoError(t, err)

	out, err := EnableProfilingForSource(store, profilesTestNamespace, "deployment", "second")

	require.NoError(t, err)
	assert.Equal(t, 2, out.ActiveSlots)
}

func TestEnableProfilingForSourceRejectsAnIncompleteIdentity(t *testing.T) {
	store := newProfilesTestStore(t)

	out, err := EnableProfilingForSource(store, "", "deployment", profilesTestWorkload)

	require.Error(t, err)
	assert.Nil(t, out)
	activeKeys, _ := store.ActiveSlots()
	assert.Empty(t, activeKeys)
}

func TestDisableProfilingForSourceRemovesTheSlotAndItsData(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	store.AddProfileData(key, otlpChunk(t, otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5))))

	out, err := DisableProfilingForSource(store, profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	assert.Equal(t, "ok", out.Status)
	assert.Equal(t, key, out.SourceKey)
	assert.Equal(t, 0, out.ActiveSlots)
	assert.False(t, store.IsActive(key))
	assert.Empty(t, store.GetProfileData(key))
}

func TestDisableProfilingForSourceRejectsAnIncompleteIdentity(t *testing.T) {
	out, err := DisableProfilingForSource(newProfilesTestStore(t), profilesTestNamespace, "", profilesTestWorkload)

	require.Error(t, err)
	assert.Nil(t, out)
}

// Clearing empties the buffer but must leave the slot open, otherwise the next batch from the
// agent would be dropped as belonging to an inactive source.
func TestClearProfilingBufferKeepsTheSlotActive(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	store.AddProfileData(key, otlpChunk(t, otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5))))
	require.NotEmpty(t, store.GetProfileData(key))

	out, err := ClearProfilingBufferForSource(store, profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	assert.Equal(t, "ok", out.Status)
	assert.Equal(t, key, out.SourceKey)
	assert.Equal(t, 1, out.ActiveSlots)
	assert.True(t, store.IsActive(key))
	assert.Empty(t, store.GetProfileData(key))
}

func TestClearProfilingBufferFailsWithoutAnActiveSlot(t *testing.T) {
	out, err := ClearProfilingBufferForSource(newProfilesTestStore(t), profilesTestNamespace, "deployment", profilesTestWorkload)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "no active profiling slot for this source")
	assert.Nil(t, out)
}

func TestClearProfilingBufferRejectsAnIncompleteIdentity(t *testing.T) {
	out, err := ClearProfilingBufferForSource(newProfilesTestStore(t), profilesTestNamespace, "deployment", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing namespace, kind, or name")
	assert.Nil(t, out)
}

func TestGetProfilingForSourceRendersTheBufferedChunks(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	store.AddProfileData(key, otlpChunk(t, otlpProfilesBatch(t,
		checkoutWorkloadProfile([]string{"main", "handler", "read"}, 4),
		checkoutWorkloadProfile([]string{"main", "handler", "write"}, 6),
	)))

	out, err := GetProfilingForSource(context.Background(), store, profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	require.NotNil(t, out.Profile.FlamebearerProfile)
	assert.Equal(t, 10, out.Profile.Flamebearer.NumTicks)
	assert.Subset(t, out.Profile.Flamebearer.Names, []string{"main", "handler", "read", "write"})

	symbolSelf := make(map[string]int64, len(out.Profile.Symbols))
	for _, s := range out.Profile.Symbols {
		symbolSelf[s.Name] = s.Self
	}
	assert.Equal(t, int64(4), symbolSelf["read"])
	assert.Equal(t, int64(6), symbolSelf["write"])
}

// Opening the profiling screen for a workload that has not reported yet must return a renderable
// empty profile, and it opens the slot so samples start accumulating.
func TestGetProfilingForSourceReturnsAnEmptyProfileAndOpensTheSlot(t *testing.T) {
	store := newProfilesTestStore(t)

	out, err := GetProfilingForSource(context.Background(), store, profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	require.NotNil(t, out.Profile.FlamebearerProfile)
	assert.Equal(t, []string{"total"}, out.Profile.Flamebearer.Names)
	assert.Equal(t, 0, out.Profile.Flamebearer.NumTicks)
	assert.Equal(t, pyroscopeMetadataFormatSingle, out.Profile.Metadata.Format)
	assert.True(t, store.IsActive("checkout-ns/Deployment/checkout-service"))
}

func TestGetProfilingForSourceRejectsAnIncompleteIdentity(t *testing.T) {
	out, err := GetProfilingForSource(context.Background(), newProfilesTestStore(t), "", "deployment", profilesTestWorkload)

	require.Error(t, err)
	assert.Nil(t, out)
}

func TestEmptyFlamebearerProfileIsRenderable(t *testing.T) {
	profile := emptyFlamebearerProfile()

	require.NotNil(t, profile.FlamebearerProfile)
	// Version 1 is the flamebearer JSON contract the Pyroscope UI component parses.
	assert.Equal(t, uint(1), profile.Version)
	assert.Equal(t, []string{"total"}, profile.Flamebearer.Names)
	assert.Equal(t, [][]int{}, profile.Flamebearer.Levels)
	assert.Equal(t, 0, profile.Flamebearer.NumTicks)
	assert.Equal(t, 0, profile.Flamebearer.MaxSelf)
	assert.Equal(t, pyroscopeMetadata(), profile.Metadata)
	assert.Empty(t, profile.Symbols)
}

// The profiling screen is per workload: one source's samples must never surface under another.
func TestProfilingSlotsAreScopedPerWorkload(t *testing.T) {
	store := newProfilesTestStore(t)
	other := checkoutWorkloadProfile([]string{"otherWorkloadFrame"}, 7)
	other.name = "other-service"

	store.EnsureSlot("checkout-ns/Deployment/other-service")
	store.AddProfileData("checkout-ns/Deployment/other-service", otlpChunk(t, otlpProfilesBatch(t, other)))

	out, err := GetProfilingForSource(context.Background(), store, profilesTestNamespace, "deployment", profilesTestWorkload)

	require.NoError(t, err)
	assert.NotContains(t, out.Profile.Flamebearer.Names, "otherWorkloadFrame")
	assert.Equal(t, 0, out.Profile.Flamebearer.NumTicks)
}
