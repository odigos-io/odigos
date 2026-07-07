package sampling

import (
	"testing"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestNoisyOperationConversionPreservesHttpQueryParams(t *testing.T) {
	value := "{ health }"
	percentage := float64(0)

	original := &v1alpha1.NoisyOperation{
		Name: "graphql health check",
		Operation: &commonapisampling.HeadSamplingOperationMatcher{
			HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
				Route:  "/graphql",
				Method: "POST",
				QueryParams: []commonapisampling.QueryParamMatcher{
					{Name: "query", ValueExact: &value},
					{Name: "operationName"},
				},
			},
		},
		PercentageAtMost: &percentage,
	}

	listed := convertNoisyOperationToModel(original)
	require.NotNil(t, listed.Operation)
	require.NotNil(t, listed.Operation.HTTPServer)
	require.Equal(t, []*model.QueryParamMatcher{
		{Name: "query", ValueExact: &value},
		{Name: "operationName"},
	}, listed.Operation.HTTPServer.QueryParams)

	updateInput := model.NoisyOperationRuleInput{
		Name:             listed.Name,
		Disabled:         &listed.Disabled,
		PercentageAtMost: listed.PercentageAtMost,
		Operation: &model.HeadSamplingOperationMatcherInput{
			HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{
				Route:       listed.Operation.HTTPServer.Route,
				RoutePrefix: listed.Operation.HTTPServer.RoutePrefix,
				Method:      listed.Operation.HTTPServer.Method,
				QueryParams: []*model.QueryParamMatcherInput{
					{Name: listed.Operation.HTTPServer.QueryParams[0].Name, ValueExact: listed.Operation.HTTPServer.QueryParams[0].ValueExact},
					{Name: listed.Operation.HTTPServer.QueryParams[1].Name, ValueExact: listed.Operation.HTTPServer.QueryParams[1].ValueExact},
				},
			},
		},
	}

	updated := noisyOperationFromInput(updateInput)
	require.NotNil(t, updated.Operation)
	require.NotNil(t, updated.Operation.HttpServer)
	require.Equal(t, original.Operation.HttpServer.QueryParams, updated.Operation.HttpServer.QueryParams)
}
