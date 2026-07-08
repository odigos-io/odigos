package sampling

import (
	"testing"

	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/frontend/graph/model"
	"github.com/stretchr/testify/require"
)

func TestHeadSamplingOperationMatcherQueryParamsRoundTrip(t *testing.T) {
	t.Parallel()

	tenant := "acme"
	original := &commonapisampling.HeadSamplingOperationMatcher{
		HttpServer: &commonapisampling.HeadSamplingHttpServerOperationMatcher{
			Route:  "/graphql",
			Method: "POST",
			QueryParams: []commonapisampling.QueryParamMatcher{
				{Name: "tenant", ValueExact: &tenant},
				{Name: "debug"},
			},
		},
	}

	graphModel := headSamplingOperationMatcherCRDToModel(original)
	require.NotNil(t, graphModel)
	require.NotNil(t, graphModel.HTTPServer)
	require.Equal(t, []*model.QueryParamMatcher{
		{Name: "tenant", ValueExact: &tenant},
		{Name: "debug"},
	}, graphModel.HTTPServer.QueryParams)

	input := &model.HeadSamplingOperationMatcherInput{
		HTTPServer: &model.HeadSamplingHTTPServerMatcherInput{
			Route:  graphModel.HTTPServer.Route,
			Method: graphModel.HTTPServer.Method,
			QueryParams: []*model.QueryParamMatcherInput{
				{Name: graphModel.HTTPServer.QueryParams[0].Name, ValueExact: graphModel.HTTPServer.QueryParams[0].ValueExact},
				{Name: graphModel.HTTPServer.QueryParams[1].Name, ValueExact: graphModel.HTTPServer.QueryParams[1].ValueExact},
			},
		},
	}

	require.Equal(t, original, headSamplingOperationMatcherInputToCRD(input))
}
