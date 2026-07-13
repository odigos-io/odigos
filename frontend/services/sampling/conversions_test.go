package sampling

import (
	"testing"

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
