package profiles

import (
	"context"
	"testing"

	"github.com/odigos-io/odigos/frontend/services/profiles/flamegraph"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pprofile"
)

func newTestProfilesConsumer(t *testing.T, store *ProfileStore, gate *IngestGate) *OdigosProfilesConsumer {
	t.Helper()
	consumer, err := NewOdigosProfilesConsumer(store, gate)
	require.NoError(t, err)
	return consumer
}

func TestOdigosProfilesConsumerDoesNotClaimToMutateData(t *testing.T) {
	consumer := newTestProfilesConsumer(t, newProfilesTestStore(t), NewProfilesIngestGate(true))

	assert.False(t, consumer.GetConsumer().Capabilities().MutatesData)
}

func TestConsumeStoresABatchForAnActiveSlot(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	err := consumer.GetConsumer().ConsumeProfiles(context.Background(),
		otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main", "handler"}, 5)))

	require.NoError(t, err)
	assert.Len(t, store.GetProfileData(key), 1)
}

// The gate is how the effective config turns profiling off without restarting the UI pod, so a
// closed gate has to drop batches before they reach the buffer.
func TestConsumeDropsEveryBatchWhileTheGateIsClosed(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	gate := NewProfilesIngestGate(false)
	consumer := newTestProfilesConsumer(t, store, gate)
	batch := otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5))

	require.NoError(t, consumer.GetConsumer().ConsumeProfiles(context.Background(), batch))
	assert.Empty(t, store.GetProfileData(key))

	gate.Set(true)
	require.NoError(t, consumer.GetConsumer().ConsumeProfiles(context.Background(), batch))
	assert.Len(t, store.GetProfileData(key), 1)
}

func TestConsumeWithoutAGateStoresTheBatch(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	consumer := newTestProfilesConsumer(t, store, nil)

	err := consumer.consume(context.Background(),
		otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5)))

	require.NoError(t, err)
	assert.Len(t, store.GetProfileData(key), 1)
}

func TestConsumeIgnoresABatchWithNoResources(t *testing.T) {
	store := newProfilesTestStore(t)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	require.NoError(t, consumer.consume(context.Background(), pprofile.NewProfiles()))

	activeKeys, keysWithData := store.ActiveSlots()
	assert.Empty(t, activeKeys)
	assert.Empty(t, keysWithData)
}

// Profiling is opt-in per workload; a batch for a source nobody asked to profile must not be
// buffered, or the UI pod would hold profile data for the whole cluster.
func TestConsumeDropsProfilesForSourcesWithoutAnOpenSlot(t *testing.T) {
	store := newProfilesTestStore(t)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	err := consumer.consume(context.Background(),
		otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5)))

	require.NoError(t, err)
	_, keysWithData := store.ActiveSlots()
	assert.Empty(t, keysWithData)
}

func TestConsumeSkipsResourcesItCannotIdentify(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	batch := otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"main"}, 5))
	// A resource with no identifying attributes at all, ahead of the identifiable one.
	unidentifiable := batch.ResourceProfiles().AppendEmpty()
	unidentifiable.ScopeProfiles().AppendEmpty().Profiles().AppendEmpty()

	require.NoError(t, consumer.consume(context.Background(), batch))

	assert.Len(t, store.GetProfileData(key), 1)
	activeKeys, _ := store.ActiveSlots()
	assert.Equal(t, []string{key}, activeKeys)
}

// One OTLP batch can carry several workloads. Each has to land in its own slot, otherwise one
// workload's flame graph would include another's stacks.
func TestConsumeSplitsABatchIntoOneChunkPerWorkload(t *testing.T) {
	store := newProfilesTestStore(t)
	checkoutKey := "checkout-ns/Deployment/checkout-service"
	paymentsKey := "checkout-ns/StatefulSet/payments-service"
	store.EnsureSlot(checkoutKey)
	store.EnsureSlot(paymentsKey)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	payments := checkoutWorkloadProfile([]string{"paymentsMain", "charge"}, 7)
	payments.kind = "StatefulSet"
	payments.name = "payments-service"

	err := consumer.consume(context.Background(), otlpProfilesBatch(t,
		checkoutWorkloadProfile([]string{"checkoutMain", "handler"}, 5),
		payments,
	))
	require.NoError(t, err)

	checkoutChunks := store.GetProfileData(checkoutKey)
	paymentsChunks := store.GetProfileData(paymentsKey)
	require.Len(t, checkoutChunks, 1)
	require.Len(t, paymentsChunks, 1)

	assert.Equal(t, []string{"checkoutMain", "handler"}, onlyStackInChunk(t, checkoutChunks[0]))
	assert.Equal(t, []string{"paymentsMain", "charge"}, onlyStackInChunk(t, paymentsChunks[0]))
}

func TestConsumeOnlyStoresTheWorkloadsWithAnOpenSlot(t *testing.T) {
	store := newProfilesTestStore(t)
	checkoutKey := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(checkoutKey)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	payments := checkoutWorkloadProfile([]string{"paymentsMain"}, 7)
	payments.kind = "StatefulSet"
	payments.name = "payments-service"

	require.NoError(t, consumer.consume(context.Background(), otlpProfilesBatch(t,
		checkoutWorkloadProfile([]string{"checkoutMain"}, 5),
		payments,
	)))

	_, keysWithData := store.ActiveSlots()
	assert.Equal(t, []string{checkoutKey}, keysWithData)
}

// OTLP profiles keep their symbols in a dictionary shared by the whole batch. A stored chunk that
// lost the dictionary would still parse, but every frame name would be unresolvable.
func TestStoredChunksCarryTheBatchDictionarySoFramesStayResolvable(t *testing.T) {
	store := newProfilesTestStore(t)
	key := "checkout-ns/Deployment/checkout-service"
	store.EnsureSlot(key)
	consumer := newTestProfilesConsumer(t, store, NewProfilesIngestGate(true))

	require.NoError(t, consumer.consume(context.Background(),
		otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"resolvableRoot", "resolvableLeaf"}, 5))))

	chunks := store.GetProfileData(key)
	require.Len(t, chunks, 1)

	req, err := flamegraph.ParseExportProfilesServiceRequest(chunks[0])
	require.NoError(t, err)
	require.NotNil(t, req.Dictionary)
	assert.NotEmpty(t, req.Dictionary.StringTable)
	assert.NotEmpty(t, req.Dictionary.StackTable)
	assert.Equal(t, []string{"resolvableRoot", "resolvableLeaf"}, onlyStackInChunk(t, chunks[0]))
}

func TestBuildSingleResourceProfilesFromBatchKeepsExactlyOneResource(t *testing.T) {
	payments := checkoutWorkloadProfile([]string{"paymentsMain"}, 7)
	payments.kind = "StatefulSet"
	payments.name = "payments-service"
	batch := otlpProfilesBatch(t, checkoutWorkloadProfile([]string{"checkoutMain"}, 5), payments)

	single := buildSingleResourceProfilesFromBatch(batch, batch.ResourceProfiles(), 1)

	require.Equal(t, 1, single.ResourceProfiles().Len())
	name, ok := single.ResourceProfiles().At(0).Resource().Attributes().Get("odigos.workload.name")
	require.True(t, ok)
	assert.Equal(t, "payments-service", name.Str())
	// The dictionary travels with the extracted resource.
	assert.Equal(t, batch.Dictionary().StringTable().Len(), single.Dictionary().StringTable().Len())
	// The source batch is left untouched.
	assert.Equal(t, 2, batch.ResourceProfiles().Len())
}

// onlyStackInChunk resolves the single sample stack in a stored chunk back to frame names, which
// only works if the chunk carries both the resource profile and the dictionary.
func onlyStackInChunk(t *testing.T, chunk []byte) []string {
	t.Helper()

	merged, _, extra := flamegraph.MergedGoogleProfileForPyroscopeSymdb([][]byte{chunk})
	require.Empty(t, extra)
	require.NotNil(t, merged)

	fb, _, err := flamegraph.BuildFlamebearerViaPyroscopeSymdb(context.Background(), [][]byte{chunk}, 2048)
	require.NoError(t, err)
	require.NotNil(t, fb)

	// The flamebearer names begin with the synthetic "total" root bar.
	require.NotEmpty(t, fb.Flamebearer.Names)
	return fb.Flamebearer.Names[1:]
}
