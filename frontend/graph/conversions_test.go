package graph

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/stretchr/testify/require"
)

func TestHeadSamplingOperationMatcherToModelPreservesQueryParams(t *testing.T) {
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
			},
		},
	}

	got := headSamplingOperationMatcherToModel(input)

	require.NotNil(t, got)
	require.NotNil(t, got.HTTPServer)
	require.Equal(t, "/graphql", *got.HTTPServer.Route)
	require.Equal(t, "POST", *got.HTTPServer.Method)
	require.Len(t, got.HTTPServer.QueryParams, 1)
	require.Equal(t, "query", got.HTTPServer.QueryParams[0].Name)
	require.NotNil(t, got.HTTPServer.QueryParams[0].ValueExact)
	require.Equal(t, valueExact, *got.HTTPServer.QueryParams[0].ValueExact)
}

func TestConvertCollectorGatewayToModelHandlesMissingServiceGraph(t *testing.T) {
	got, err := convertCollectorGatewayToModel(&common.CollectorGatewayConfiguration{}, newProvenanceCollector(nil))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.Nil(t, got.ServiceGraphDisabled)
}

func TestConvertCollectorGatewayToModelPreservesServiceGraphDisabled(t *testing.T) {
	disabled := true

	got, err := convertCollectorGatewayToModel(&common.CollectorGatewayConfiguration{
		ServiceGraph: &common.ServiceGraphOptions{
			Disabled: &disabled,
		},
	}, newProvenanceCollector(nil))

	require.NoError(t, err)
	require.NotNil(t, got)
	require.NotNil(t, got.ServiceGraphDisabled)
	require.True(t, *got.ServiceGraphDisabled)
}
