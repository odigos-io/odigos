package sampling

import (
	"testing"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestHeadSamplingOperationMatcherInputToCRDPreservesQueryParams(t *testing.T) {
	valueExact := "{ health }"
	input := &model.HeadSamplingOperationMatcherInput{
		HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{
			Route:  stringPtr("/graphql"),
			Method: stringPtr("POST"),
			QueryParams: []*model.HeadSamplingQueryParamMatcherInput{
				{
					Name:       "query",
					ValueExact: &valueExact,
				},
				{
					Name: "operationName",
				},
			},
		},
	}

	got := headSamplingOperationMatcherInputToCRD(input)

	require.NotNil(t, got)
	require.NotNil(t, got.HttpServer)
	require.Equal(t, "/graphql", got.HttpServer.Route)
	require.Equal(t, "POST", got.HttpServer.Method)
	require.Len(t, got.HttpServer.QueryParams, 2)
	require.Equal(t, "query", got.HttpServer.QueryParams[0].Name)
	require.NotNil(t, got.HttpServer.QueryParams[0].ValueExact)
	require.Equal(t, valueExact, *got.HttpServer.QueryParams[0].ValueExact)
	require.Equal(t, "operationName", got.HttpServer.QueryParams[1].Name)
	require.Nil(t, got.HttpServer.QueryParams[1].ValueExact)
}

func TestHeadSamplingOperationMatcherCRDToModelPreservesQueryParams(t *testing.T) {
	valueExact := "{ health }"
	input := &commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
			Route:  "/graphql",
			Method: "POST",
			QueryParams: []commonapisampling.QueryParamMatcher{
				{
					Name:       "query",
					ValueExact: &valueExact,
				},
				{
					Name: "operationName",
				},
			},
		},
	}

	got := headSamplingOperationMatcherCRDToModel(input)

	require.NotNil(t, got)
	require.NotNil(t, got.HTTPServer)
	require.Equal(t, "/graphql", *got.HTTPServer.Route)
	require.Equal(t, "POST", *got.HTTPServer.Method)
	require.Len(t, got.HTTPServer.QueryParams, 2)
	require.Equal(t, "query", got.HTTPServer.QueryParams[0].Name)
	require.NotNil(t, got.HTTPServer.QueryParams[0].ValueExact)
	require.Equal(t, valueExact, *got.HTTPServer.QueryParams[0].ValueExact)
	require.Equal(t, "operationName", got.HTTPServer.QueryParams[1].Name)
	require.Nil(t, got.HTTPServer.QueryParams[1].ValueExact)
}

func stringPtr(value string) *string {
	return &value
}

func boolPtr(value bool) *bool {
	return &value
}

func float64Ptr(value float64) *float64 {
	return &value
}

func intPtr(value int) *int {
	return &value
}

func TestMergeNoisyOperationUpdatePreservesScopesAndOperationOnOmit(t *testing.T) {
	existing := v1alpha1.NoisyOperation{
		Name:     "payments-health",
		Disabled: true,
		SourceScopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"payments"},
		},
		Operation: &commonapisampling.HeadSamplingOperationMatcher{
			HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
				Route:  "/healthz",
				Method: "GET",
			},
		},
		PercentageAtMost: float64Ptr(5),
		Notes:            "scoped drop",
	}

	got := mergeNoisyOperationUpdate(existing, model.NoisyOperationRuleInput{
		Name: stringPtr("renamed"),
	})

	require.Equal(t, "renamed", got.Name)
	require.True(t, got.Disabled)
	require.Equal(t, existing.SourceScopes, got.SourceScopes)
	require.Equal(t, existing.Operation, got.Operation)
	require.Equal(t, existing.PercentageAtMost, got.PercentageAtMost)
	require.Equal(t, "scoped drop", got.Notes)
}

func TestMergeNoisyOperationUpdateEmptyScopesClearsExplicitly(t *testing.T) {
	existing := v1alpha1.NoisyOperation{
		SourceScopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"payments"},
		},
		Operation: &commonapisampling.HeadSamplingOperationMatcher{
			HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
				Route: "/healthz",
			},
		},
		PercentageAtMost: float64Ptr(5),
	}

	got := mergeNoisyOperationUpdate(existing, model.NoisyOperationRuleInput{
		SourceScopes:     &model.SourcesScopesInput{},
		PercentageAtMost: float64Ptr(10),
	})

	require.NotNil(t, got.SourceScopes)
	require.Empty(t, got.SourceScopes.Namespaces)
	require.Equal(t, existing.Operation, got.Operation)
	require.Equal(t, float64Ptr(10), got.PercentageAtMost)
}

func TestMergeCostReductionRuleUpdatePreservesScopesAndOperationOnOmit(t *testing.T) {
	existing := v1alpha1.CostReductionRule{
		Name: "checkout-drop",
		SourceScopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"checkout"},
		},
		Operation: &commonapisampling.TailSamplingOperationMatcher{
			HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{
				Route: "/api/checkout",
			},
		},
		PercentageAtMost: 0,
		Notes:            "keep scoped",
	}

	got := mergeCostReductionRuleUpdate(existing, model.CostReductionRuleInput{
		Name:             stringPtr("renamed"),
		PercentageAtMost: 1,
	})

	require.Equal(t, "renamed", got.Name)
	require.Equal(t, existing.SourceScopes, got.SourceScopes)
	require.Equal(t, existing.Operation, got.Operation)
	require.Equal(t, 1.0, got.PercentageAtMost)
	require.Equal(t, "keep scoped", got.Notes)
}

func TestMergeHighlyRelevantOperationUpdatePreservesMatchersOnOmit(t *testing.T) {
	existing := v1alpha1.HighlyRelevantOperation{
		Name:     "keep-errors",
		Disabled: false,
		SourceScopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"payments"},
		},
		Error:             true,
		DurationAtLeastMs: intPtr(500),
		Operation: &commonapisampling.TailSamplingOperationMatcher{
			HttpServer: &commonapisampling.TailSamplingHttpServerOperationMatcher{
				Route: "/charge",
			},
		},
		PercentageAtLeast: float64Ptr(100),
		Notes:             "keep scoped errors",
	}

	got := mergeHighlyRelevantOperationUpdate(existing, model.HighlyRelevantOperationRuleInput{
		Name: stringPtr("renamed"),
	})

	require.Equal(t, "renamed", got.Name)
	require.False(t, got.Disabled)
	require.Equal(t, existing.SourceScopes, got.SourceScopes)
	require.True(t, got.Error)
	require.Equal(t, existing.DurationAtLeastMs, got.DurationAtLeastMs)
	require.Equal(t, existing.Operation, got.Operation)
	require.Equal(t, existing.PercentageAtLeast, got.PercentageAtLeast)
	require.Equal(t, "keep scoped errors", got.Notes)
}

func TestMergeHighlyRelevantOperationUpdateCanClearErrorExplicitly(t *testing.T) {
	existing := v1alpha1.HighlyRelevantOperation{
		Error: true,
		SourceScopes: &k8sconsts.SourcesScopes{
			Namespaces: []string{"payments"},
		},
	}

	got := mergeHighlyRelevantOperationUpdate(existing, model.HighlyRelevantOperationRuleInput{
		Error: boolPtr(false),
	})

	require.False(t, got.Error)
	require.Equal(t, existing.SourceScopes, got.SourceScopes)
}
