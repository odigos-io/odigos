package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func payloadCollectionDistro() *distro.OtelDistro {
	return &distro.OtelDistro{
		Traces: &distro.Traces{
			PayloadCollection: &distro.PayloadCollection{Supported: true},
		},
	}
}

func int64Ptr(v int64) *int64 { return &v }

func strSlicePtr(v ...string) *[]string { return &v }

// A rule that applies to every container in the workload.
func clusterWideRule() odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			PayloadCollection: &instrumentationrules.PayloadCollection{
				DbQuery: &instrumentationrules.DbQueryPayloadCollection{
					MaxPayloadLength: int64Ptr(4096),
				},
			},
		},
	}
}

// A rule that only applies to some of the containers, and which turns on
// collection of raw http request bodies.
func httpBodyRule() odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			PayloadCollection: &instrumentationrules.PayloadCollection{
				HttpRequest: &instrumentationrules.HttpPayloadCollection{
					MimeTypes:        strSlicePtr("application/json"),
					MaxPayloadLength: int64Ptr(1024),
				},
			},
		},
	}
}

func TestCalculatePayloadCollectionConfig_doesNotMutateInputRules(t *testing.T) {
	d := payloadCollectionDistro()

	shared := clusterWideRule()
	scoped := httpBodyRule()

	// The container that both rules apply to.
	both := []odigosv1.InstrumentationRule{shared, scoped}
	merged := CalculatePayloadCollectionConfig(d, &both)
	require.NotNil(t, merged)
	require.NotNil(t, merged.HttpRequest, "the merged config for the targeted container collects http bodies")

	// The rules the caller passed in must be unchanged, so that the next
	// container can be evaluated against the rules the user actually wrote.
	require.Nil(t, shared.Spec.PayloadCollection.HttpRequest,
		"merging must not write the scoped rule's http payload collection into the cluster-wide rule")
}

func TestCalculatePayloadCollectionConfig_scopedRuleDoesNotLeakToOtherContainer(t *testing.T) {
	d := payloadCollectionDistro()

	shared := clusterWideRule()
	scoped := httpBodyRule()

	// Container A is in scope for both rules.
	containerA := []odigosv1.InstrumentationRule{shared, scoped}
	gotA := CalculatePayloadCollectionConfig(d, &containerA)
	require.NotNil(t, gotA.HttpRequest)

	// Container B (e.g. a sidecar in another language) is only in scope for the
	// cluster-wide rule, which never asked for http bodies to be collected.
	containerB := []odigosv1.InstrumentationRule{shared}
	gotB := CalculatePayloadCollectionConfig(d, &containerB)

	require.NotNil(t, gotB)
	require.Nil(t, gotB.HttpRequest,
		"a container that is out of scope for the http rule must not collect http request bodies")
	require.Equal(t, int64(4096), *gotB.DbQuery.MaxPayloadLength)
}

func TestCalculatePayloadCollectionConfig_mimeTypesOrderIsDeterministic(t *testing.T) {
	d := payloadCollectionDistro()

	build := func() *[]odigosv1.InstrumentationRule {
		rules := []odigosv1.InstrumentationRule{
			mimeRule("application/json", "text/plain"),
			mimeRule("application/xml", "text/html"),
		}
		return &rules
	}

	first := CalculatePayloadCollectionConfig(d, build())
	require.NotNil(t, first.HttpRequest)
	require.ElementsMatch(t,
		[]string{"application/json", "text/plain", "application/xml", "text/html"},
		*first.HttpRequest.MimeTypes)

	// The merged config is written into the InstrumentationConfig spec, so an
	// unstable order means a new spec revision on every reconcile.
	for i := 0; i < 50; i++ {
		got := CalculatePayloadCollectionConfig(d, build())
		require.Equal(t, *first.HttpRequest.MimeTypes, *got.HttpRequest.MimeTypes,
			"merged mime types must have a stable order across calls")
	}
}

func TestCalculatePayloadCollectionConfig_mergesRestrictiveValues(t *testing.T) {
	d := payloadCollectionDistro()

	rules := []odigosv1.InstrumentationRule{
		{Spec: odigosv1.InstrumentationRuleSpec{
			PayloadCollection: &instrumentationrules.PayloadCollection{
				HttpRequest: &instrumentationrules.HttpPayloadCollection{
					MaxPayloadLength:    int64Ptr(4096),
					DropPartialPayloads: boolPtr(false),
				},
			},
		}},
		{Spec: odigosv1.InstrumentationRuleSpec{
			PayloadCollection: &instrumentationrules.PayloadCollection{
				HttpRequest: &instrumentationrules.HttpPayloadCollection{
					MaxPayloadLength:    int64Ptr(1024),
					DropPartialPayloads: boolPtr(true),
				},
			},
		}},
	}

	got := CalculatePayloadCollectionConfig(d, &rules)

	require.Equal(t, int64(1024), *got.HttpRequest.MaxPayloadLength)
	require.True(t, *got.HttpRequest.DropPartialPayloads)
}

func mimeRule(mimeTypes ...string) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{
			PayloadCollection: &instrumentationrules.PayloadCollection{
				HttpRequest: &instrumentationrules.HttpPayloadCollection{
					MimeTypes: strSlicePtr(mimeTypes...),
				},
			},
		},
	}
}
