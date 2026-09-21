package profiles

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/pdata/pcommon"
	semconv "go.opentelemetry.io/otel/semconv/v1.34.0"
)

func TestSourceKeyFromSourceIDJoinsNamespaceKindAndName(t *testing.T) {
	key := SourceKeyFromSourceID(common.SourceID{
		Namespace: profilesTestNamespace,
		Kind:      k8sconsts.WorkloadKindDeployment,
		Name:      profilesTestWorkload,
	})

	assert.Equal(t, "checkout-ns/Deployment/checkout-service", key)
}

func TestSourceKeyFromResourceBuildsTheSameKeyAsTheSourceID(t *testing.T) {
	attrs := pcommon.NewMap()
	attrs.PutStr(string(semconv.K8SNamespaceNameKey), profilesTestNamespace)
	attrs.PutStr(consts.OdigosWorkloadKindAttribute, "Deployment")
	attrs.PutStr(consts.OdigosWorkloadNameAttribute, profilesTestWorkload)

	key, ok := SourceKeyFromResource(attrs)

	require.True(t, ok)
	assert.Equal(t, "checkout-ns/Deployment/checkout-service", key)
}

// The slot key is written by the OTLP consumer from resource attributes and read by the GraphQL
// layer from strings the UI sends. If the two disagree for any workload kind, the profiling
// screen silently shows an empty flame graph for that kind.
func TestTheConsumerAndTheGraphqlLayerAgreeOnTheSlotKey(t *testing.T) {
	kinds := []k8sconsts.WorkloadKind{
		k8sconsts.WorkloadKindDeployment,
		k8sconsts.WorkloadKindDaemonSet,
		k8sconsts.WorkloadKindStatefulSet,
		k8sconsts.WorkloadKindCronJob,
		k8sconsts.WorkloadKindJob,
		k8sconsts.WorkloadKindStaticPod,
		k8sconsts.WorkloadKindArgoRollout,
		k8sconsts.WorkloadKindDeploymentConfig,
	}

	for _, kind := range kinds {
		t.Run(string(kind), func(t *testing.T) {
			attrs := pcommon.NewMap()
			attrs.PutStr(string(semconv.K8SNamespaceNameKey), profilesTestNamespace)
			attrs.PutStr(consts.OdigosWorkloadKindAttribute, string(kind))
			attrs.PutStr(consts.OdigosWorkloadNameAttribute, profilesTestWorkload)

			ingestKey, ok := SourceKeyFromResource(attrs)
			require.True(t, ok, "consumer could not derive a key for %s", kind)

			// The UI may send either casing for the kind.
			for _, sent := range []string{string(kind), lowerFirst(string(kind))} {
				id, err := SourceIDFromStrings(profilesTestNamespace, sent, profilesTestWorkload)
				require.NoError(t, err)
				assert.Equal(t, ingestKey, SourceKeyFromSourceID(id), "kind sent as %q", sent)
			}
		})
	}
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	b := []byte(s)
	if b[0] >= 'A' && b[0] <= 'Z' {
		b[0] += 'a' - 'A'
	}
	return string(b)
}

func TestSourceKeyFromResourceRejectsResourcesItCannotIdentify(t *testing.T) {
	t.Run("no namespace", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr(consts.OdigosWorkloadKindAttribute, "Deployment")
		attrs.PutStr(consts.OdigosWorkloadNameAttribute, profilesTestWorkload)

		_, ok := SourceKeyFromResource(attrs)
		assert.False(t, ok)
	})

	t.Run("no workload name", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), profilesTestNamespace)
		attrs.PutStr(consts.OdigosWorkloadKindAttribute, "Deployment")

		_, ok := SourceKeyFromResource(attrs)
		assert.False(t, ok)
	})

	t.Run("empty attributes", func(t *testing.T) {
		_, ok := SourceKeyFromResource(pcommon.NewMap())
		assert.False(t, ok)
	})

	// The attribute is present but blank, which resolves without error and would otherwise
	// produce a nameless slot key that swallows data.
	t.Run("blank workload name", func(t *testing.T) {
		attrs := pcommon.NewMap()
		attrs.PutStr(string(semconv.K8SNamespaceNameKey), profilesTestNamespace)
		attrs.PutStr(consts.OdigosWorkloadKindAttribute, "Deployment")
		attrs.PutStr(consts.OdigosWorkloadNameAttribute, "")

		_, ok := SourceKeyFromResource(attrs)
		assert.False(t, ok)
	})
}

func TestNormalizeWorkloadKindCanonicalizesKnownKinds(t *testing.T) {
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, NormalizeWorkloadKind("deployment"))
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, NormalizeWorkloadKind("Deployment"))
	assert.Equal(t, k8sconsts.WorkloadKindDeployment, NormalizeWorkloadKind("DEPLOYMENT"))
	assert.Equal(t, k8sconsts.WorkloadKindCronJob, NormalizeWorkloadKind("cronjob"))
	assert.Equal(t, k8sconsts.WorkloadKindStatefulSet, NormalizeWorkloadKind("statefulset"))
}

func TestNormalizeWorkloadKindPassesAnUnknownKindThrough(t *testing.T) {
	assert.Equal(t, k8sconsts.WorkloadKind("Sidecar"), NormalizeWorkloadKind("Sidecar"))
	assert.Equal(t, k8sconsts.WorkloadKind(""), NormalizeWorkloadKind(""))
}

func TestEarliestProfileStartTimeUsesTheSmallestTimestampInSeconds(t *testing.T) {
	early := checkoutWorkloadProfile([]string{"main"}, 5)
	early.timeNano = 1_700_000_002_000_000_000
	late := checkoutWorkloadProfile([]string{"main"}, 5)
	late.timeNano = 1_700_000_009_000_000_000

	seconds := earliestProfileStartTimeUnixSec([][]byte{
		otlpChunk(t, otlpProfilesBatch(t, late)),
		otlpChunk(t, otlpProfilesBatch(t, early)),
	})

	assert.Equal(t, int64(1_700_000_002), seconds)
}

// An unstamped profile must be ignored whichever side of a stamped one it arrives on, or the
// timeline would start at the epoch.
func TestEarliestProfileStartTimeIgnoresUnstampedProfiles(t *testing.T) {
	unstamped := checkoutWorkloadProfile([]string{"main"}, 5)
	unstamped.timeNano = 0
	stamped := checkoutWorkloadProfile([]string{"main"}, 5)
	stamped.timeNano = 1_700_000_005_000_000_000

	t.Run("unstamped first", func(t *testing.T) {
		seconds := earliestProfileStartTimeUnixSec([][]byte{
			otlpChunk(t, otlpProfilesBatch(t, unstamped, stamped)),
		})
		assert.Equal(t, int64(1_700_000_005), seconds)
	})

	t.Run("unstamped last", func(t *testing.T) {
		seconds := earliestProfileStartTimeUnixSec([][]byte{
			otlpChunk(t, otlpProfilesBatch(t, stamped, unstamped)),
		})
		assert.Equal(t, int64(1_700_000_005), seconds)
	})
}

func TestEarliestProfileStartTimeSkipsChunksItCannotParse(t *testing.T) {
	stamped := checkoutWorkloadProfile([]string{"main"}, 5)
	stamped.timeNano = 1_700_000_005_000_000_000

	seconds := earliestProfileStartTimeUnixSec([][]byte{
		[]byte("not a protobuf message"),
		otlpChunk(t, otlpProfilesBatch(t, stamped)),
	})

	assert.Equal(t, int64(1_700_000_005), seconds)
}

func TestEarliestProfileStartTimeWithoutAnyUsableChunkIsZero(t *testing.T) {
	assert.Equal(t, int64(0), earliestProfileStartTimeUnixSec(nil))
	assert.Equal(t, int64(0), earliestProfileStartTimeUnixSec([][]byte{[]byte("garbage")}))
}

// Sub-second timestamps truncate to 0 rather than rounding up into the future.
func TestEarliestProfileStartTimeTruncatesSubSecondTimestamps(t *testing.T) {
	subSecond := checkoutWorkloadProfile([]string{"main"}, 5)
	subSecond.timeNano = 999_999_999

	seconds := earliestProfileStartTimeUnixSec([][]byte{otlpChunk(t, otlpProfilesBatch(t, subSecond))})

	assert.Equal(t, int64(0), seconds)
}
