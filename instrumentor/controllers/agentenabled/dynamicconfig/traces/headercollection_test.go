package traces

import (
	"testing"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/api/instrumentationrules"
	"github.com/odigos-io/odigos/distros/distro"
	"github.com/stretchr/testify/require"
)

func headerCollectionRule(hc *instrumentationrules.HttpHeadersCollection) odigosv1.InstrumentationRule {
	return odigosv1.InstrumentationRule{
		Spec: odigosv1.InstrumentationRuleSpec{HeadersCollection: hc},
	}
}

var headersCollectionSupportingDistro = distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
	HeadersCollection: &distro.HeadersCollection{Supported: true},
})

func TestDistroSupportsTracesHeadersCollection(t *testing.T) {
	require.False(t, DistroSupportsTracesHeadersCollection(distroWithTraces(common.JavaProgrammingLanguage, nil)))
	require.False(t, DistroSupportsTracesHeadersCollection(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{})))
	require.False(t, DistroSupportsTracesHeadersCollection(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		HeadersCollection: &distro.HeadersCollection{Supported: false},
	})))
	require.True(t, DistroSupportsTracesHeadersCollection(headersCollectionSupportingDistro))
}

// Headers routinely carry credentials, so a distro that does not support the feature must not be
// asked to collect them.
func TestCalculateHeaderCollectionConfig_unsupportedDistro(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{headerCollectionRule(&instrumentationrules.HttpHeadersCollection{
		HeaderKeys: []string{"x-request-id"},
	})}

	require.Nil(t, CalculateHeaderCollectionConfig(distroWithTraces(common.JavaProgrammingLanguage, &distro.Traces{
		HeadersCollection: &distro.HeadersCollection{Supported: false},
	}), &irls))
}

func TestCalculateHeaderCollectionConfig_noRules(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{}
	require.Nil(t, CalculateHeaderCollectionConfig(headersCollectionSupportingDistro, &irls))
}

func TestCalculateHeaderCollectionConfig_ruleWithoutHeaderKeys(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		headerCollectionRule(nil),
		headerCollectionRule(&instrumentationrules.HttpHeadersCollection{HeaderKeys: []string{}}),
	}

	require.Nil(t, CalculateHeaderCollectionConfig(headersCollectionSupportingDistro, &irls))
}

func TestCalculateHeaderCollectionConfig_singleRule(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{headerCollectionRule(&instrumentationrules.HttpHeadersCollection{
		HeaderKeys: []string{"x-request-id", "x-tenant"},
	})}

	got := CalculateHeaderCollectionConfig(headersCollectionSupportingDistro, &irls)

	require.NotNil(t, got)
	require.Equal(t, []string{"x-request-id", "x-tenant"}, got.HeaderKeys)
}

func TestCalculateHeaderCollectionConfig_concatenatesAllRules(t *testing.T) {
	irls := []odigosv1.InstrumentationRule{
		headerCollectionRule(&instrumentationrules.HttpHeadersCollection{HeaderKeys: []string{"x-request-id"}}),
		headerCollectionRule(nil),
		headerCollectionRule(&instrumentationrules.HttpHeadersCollection{HeaderKeys: []string{"x-tenant", "x-request-id"}}),
	}

	got := CalculateHeaderCollectionConfig(headersCollectionSupportingDistro, &irls)

	require.NotNil(t, got)
	require.Equal(t, []string{"x-request-id", "x-tenant", "x-request-id"}, got.HeaderKeys)
}
