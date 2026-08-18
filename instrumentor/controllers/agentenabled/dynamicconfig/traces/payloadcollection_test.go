package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

// newPtr returns a pointer to v, for the many optional fields in the instrumentation rules API.
func newPtr[T any](v T) *T {
	return &v
}

// distroWithTraces builds a minimal distro carrying only the traces capabilities under test.
func distroWithTraces(language common.ProgrammingLanguage, traces *distro.Traces) *distro.OtelDistro {
	return &distro.OtelDistro{
		Name:     "test-distro",
		Language: language,
		Traces:   traces,
	}
}

func payloadCollectionRule(pc *instrumentationrules.PayloadCollection) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{PayloadCollection: pc},
	}
}

var payloadCollectionSupportingDistro = distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
	PayloadCollection: &distro.PayloadCollection{Supported: true},
})

func TestDistroSupportsTracesPayloadCollection(t *testing.T) {
	require.False(t, DistroSupportsTracesPayloadCollection(distroWithTraces(common.JavaProgrammingLanguage, nil)))
	require.False(t, DistroSupportsTracesPayloadCollection(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{})))
	require.False(t, DistroSupportsTracesPayloadCollection(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		PayloadCollection: &distro.PayloadCollection{Supported: false},
	})))
	require.True(t, DistroSupportsTracesPayloadCollection(payloadCollectionSupportingDistro))
}

// A distro that cannot collect payloads must never be handed a payload collection config, even
// when rules ask for it.
func TestCalculatePayloadCollectionConfig_unsupportedDistro(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{payloadCollectionRule(&instrumentationrules.PayloadCollection{
		HttpRequest: &instrumentationrules.HttpPayloadCollection{},
	})}

	got := CalculatePayloadCollectionConfig(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		PayloadCollection: &distro.PayloadCollection{Supported: false},
	}), &irls)

	require.Nil(t, got)
}

func TestCalculatePayloadCollectionConfig_noRules(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{}
	require.Nil(t, CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls))
}

func TestCalculatePayloadCollectionConfig_rulesWithoutPayloadCollection(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{payloadCollectionRule(nil), payloadCollectionRule(nil)}
	require.Nil(t, CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls))
}

// Rules that configure other features are in the same list; they must not clear what an earlier
// payload collection rule established.
func TestCalculatePayloadCollectionConfig_unrelatedRuleDoesNotClearTheConfig(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			HttpRequest: &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(1024))},
		}),
		payloadCollectionRule(nil),
	}

	got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

	require.NotNil(t, got)
	require.Equal(t, int64(1024), *got.HttpRequest.MaxPayloadLength)
}

func TestCalculatePayloadCollectionConfig_singleRulePassedThrough(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{payloadCollectionRule(&instrumentationrules.PayloadCollection{
		HttpRequest: &instrumentationrules.HttpPayloadCollection{
			MimeTypes:        newPtr([]string{"application/json"}),
			MaxPayloadLength: newPtr(int64(1024)),
		},
	})}

	got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

	require.NotNil(t, got)
	require.NotNil(t, got.HttpRequest)
	require.Equal(t, []string{"application/json"}, *got.HttpRequest.MimeTypes)
	require.Equal(t, int64(1024), *got.HttpRequest.MaxPayloadLength)
	require.Nil(t, got.HttpResponse)
	require.Nil(t, got.DbQuery)
	require.Nil(t, got.Messaging)
}

// Each payload kind is merged independently, so a rule that only configures one of them must not
// wipe the others.
func TestCalculatePayloadCollectionConfig_mergesDisjointPayloadKinds(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			HttpRequest: &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(100))},
		}),
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			HttpResponse: &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(200))},
		}),
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			DbQuery: &instrumentationrules.DbQueryPayloadCollection{MaxPayloadLength: newPtr(int64(300))},
		}),
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			Messaging: &instrumentationrules.MessagingPayloadCollection{MaxPayloadLength: newPtr(int64(400))},
		}),
	}

	got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

	require.NotNil(t, got)
	require.Equal(t, int64(100), *got.HttpRequest.MaxPayloadLength)
	require.Equal(t, int64(200), *got.HttpResponse.MaxPayloadLength)
	require.Equal(t, int64(300), *got.DbQuery.MaxPayloadLength)
	require.Equal(t, int64(400), *got.Messaging.MaxPayloadLength)
}

// The merge is documented to take the most restrictive value across rules, which is what keeps a
// rule that caps payload size from being undone by a laxer one.
func TestCalculatePayloadCollectionConfig_takesTheMostRestrictiveValues(t *testing.T) {
	tt := []struct {
		name               string
		first              *instrumentationrules.HttpPayloadCollection
		second             *instrumentationrules.HttpPayloadCollection
		expectedMaxLength  *int64
		expectedDropShared *bool
	}{
		{
			name:              "lower max payload length wins when it comes second",
			first:             &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(2048))},
			second:            &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(512))},
			expectedMaxLength: newPtr(int64(512)),
		},
		{
			name:              "lower max payload length wins when it comes first",
			first:             &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(512))},
			second:            &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(2048))},
			expectedMaxLength: newPtr(int64(512)),
		},
		{
			name:              "unset max payload length takes the other value",
			first:             &instrumentationrules.HttpPayloadCollection{},
			second:            &instrumentationrules.HttpPayloadCollection{MaxPayloadLength: newPtr(int64(512))},
			expectedMaxLength: newPtr(int64(512)),
		},
		{
			name:               "drop partial payloads wins over keep",
			first:              &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(false)},
			second:             &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(true)},
			expectedDropShared: newPtr(true),
		},
		{
			name:               "drop partial payloads wins regardless of order",
			first:              &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(true)},
			second:             &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(false)},
			expectedDropShared: newPtr(true),
		},
		{
			name:               "both keep partial payloads",
			first:              &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(false)},
			second:             &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(false)},
			expectedDropShared: newPtr(false),
		},
		{
			name:               "unset drop partial payloads takes the other value",
			first:              &instrumentationrules.HttpPayloadCollection{},
			second:             &instrumentationrules.HttpPayloadCollection{DropPartialPayloads: newPtr(true)},
			expectedDropShared: newPtr(true),
		},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			irls := []odigosv1.InstrumentationRule{
				payloadCollectionRule(&instrumentationrules.PayloadCollection{HttpRequest: tc.first}),
				payloadCollectionRule(&instrumentationrules.PayloadCollection{HttpRequest: tc.second}),
			}

			got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

			require.NotNil(t, got)
			require.NotNil(t, got.HttpRequest)
			require.Equal(t, tc.expectedMaxLength, got.HttpRequest.MaxPayloadLength)
			require.Equal(t, tc.expectedDropShared, got.HttpRequest.DropPartialPayloads)
		})
	}
}

// Mime types are a union: an allow-list from one rule must not silence another rule's types.
func TestCalculatePayloadCollectionConfig_unionsMimeTypes(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			HttpRequest: &instrumentationrules.HttpPayloadCollection{
				MimeTypes: newPtr([]string{"application/json", "text/plain"}),
			},
		}),
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			HttpRequest: &instrumentationrules.HttpPayloadCollection{
				MimeTypes: newPtr([]string{"application/json", "application/xml"}),
			},
		}),
	}

	got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

	require.NotNil(t, got.HttpRequest.MimeTypes)
	// the merge de-duplicates through a map, so the order is not stable
	require.ElementsMatch(t, []string{"application/json", "text/plain", "application/xml"}, *got.HttpRequest.MimeTypes)
}

func TestCalculatePayloadCollectionConfig_mergesMessagingAndDbLimits(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			DbQuery:   &instrumentationrules.DbQueryPayloadCollection{MaxPayloadLength: newPtr(int64(900)), DropPartialPayloads: newPtr(false)},
			Messaging: &instrumentationrules.MessagingPayloadCollection{MaxPayloadLength: newPtr(int64(900)), DropPartialPayloads: newPtr(false)},
		}),
		payloadCollectionRule(&instrumentationrules.PayloadCollection{
			DbQuery:   &instrumentationrules.DbQueryPayloadCollection{MaxPayloadLength: newPtr(int64(100)), DropPartialPayloads: newPtr(true)},
			Messaging: &instrumentationrules.MessagingPayloadCollection{MaxPayloadLength: newPtr(int64(100)), DropPartialPayloads: newPtr(true)},
		}),
	}

	got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

	require.Equal(t, int64(100), *got.DbQuery.MaxPayloadLength)
	require.True(t, *got.DbQuery.DropPartialPayloads)
	require.Equal(t, int64(100), *got.Messaging.MaxPayloadLength)
	require.True(t, *got.Messaging.DropPartialPayloads)
}

// The sanitization policy decides whether raw DB queries (which can hold PII) reach the backend,
// so the strictest policy across rules has to win in both orders.
func TestCalculatePayloadCollectionConfig_dbQuerySanitizationPolicy(t *testing.T) {
	sanitized := consts.DbQuerySanitizationPolicySanitized
	sanitizedOrFull := consts.DbQuerySanitizationPolicySanitizedOrFull
	full := consts.DbQuerySanitizationPolicyFull
	unknown := consts.DbQuerySanitizationPolicy("something-else")

	tt := []struct {
		name     string
		first    *consts.DbQuerySanitizationPolicy
		second   *consts.DbQuerySanitizationPolicy
		expected *consts.DbQuerySanitizationPolicy
	}{
		{name: "both unset", first: nil, second: nil, expected: nil},
		{name: "only the second is set", first: nil, second: &full, expected: &full},
		{name: "only the first is set", first: &full, second: nil, expected: &full},
		{name: "sanitized beats full", first: &full, second: &sanitized, expected: &sanitized},
		{name: "sanitized beats full in reverse order", first: &sanitized, second: &full, expected: &sanitized},
		{name: "sanitized beats sanitized-or-full", first: &sanitizedOrFull, second: &sanitized, expected: &sanitized},
		{name: "sanitized-or-full beats full", first: &full, second: &sanitizedOrFull, expected: &sanitizedOrFull},
		{name: "equal policies keep the value", first: &sanitized, second: &sanitized, expected: &sanitized},
		{name: "any known policy beats an unrecognized one", first: &unknown, second: &full, expected: &full},
	}

	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			irls := []odigosv1.InstrumentationRule{
				payloadCollectionRule(&instrumentationrules.PayloadCollection{
					DbQuery: &instrumentationrules.DbQueryPayloadCollection{SanitizationPolicy: tc.first},
				}),
				payloadCollectionRule(&instrumentationrules.PayloadCollection{
					DbQuery: &instrumentationrules.DbQueryPayloadCollection{SanitizationPolicy: tc.second},
				}),
			}

			got := CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls)

			require.NotNil(t, got.DbQuery)
			if tc.expected == nil {
				require.Nil(t, got.DbQuery.SanitizationPolicy)
			} else {
				require.Equal(t, *tc.expected, *got.DbQuery.SanitizationPolicy)
			}
		})
	}
}

// Three rules are the realistic case (a cluster-wide rule plus two scoped ones) and the result
// must not depend on which of them the reconcile happened to list first.
func TestCalculatePayloadCollectionConfig_mergeIsOrderIndependent(t *testing.T) {
	build := func() []*instrumentationrules.PayloadCollection {
		return []*instrumentationrules.PayloadCollection{
			{HttpRequest: &instrumentationrules.HttpPayloadCollection{
				MaxPayloadLength:    newPtr(int64(4096)),
				DropPartialPayloads: newPtr(false),
				MimeTypes:           newPtr([]string{"text/plain"}),
			}},
			{HttpRequest: &instrumentationrules.HttpPayloadCollection{
				MaxPayloadLength: newPtr(int64(256)),
				MimeTypes:        newPtr([]string{"application/json"}),
			}},
			{HttpRequest: &instrumentationrules.HttpPayloadCollection{
				DropPartialPayloads: newPtr(true),
			}},
		}
	}

	assertMerged := func(t *testing.T, got *instrumentationrules.PayloadCollection) {
		t.Helper()
		require.NotNil(t, got)
		require.Equal(t, int64(256), *got.HttpRequest.MaxPayloadLength)
		require.True(t, *got.HttpRequest.DropPartialPayloads)
		require.ElementsMatch(t, []string{"text/plain", "application/json"}, *got.HttpRequest.MimeTypes)
	}

	forward := build()
	irls := []odigosv1.InstrumentationRule{
		payloadCollectionRule(forward[0]),
		payloadCollectionRule(forward[1]),
		payloadCollectionRule(forward[2]),
	}
	assertMerged(t, CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls))

	reversed := build()
	irls = []odigosv1.InstrumentationRule{
		payloadCollectionRule(reversed[2]),
		payloadCollectionRule(reversed[1]),
		payloadCollectionRule(reversed[0]),
	}
	assertMerged(t, CalculatePayloadCollectionConfig(payloadCollectionSupportingDistro, &irls))
}
