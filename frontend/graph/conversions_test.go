package graph

import (
	"testing"

	"github.com/odigos-io/odigos/common"
	commonapisampling "github.com/odigos-io/odigos/common/api/sampling"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services"
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

func TestEffectiveConfigToModelInterrogationAPIKeyProvenance(t *testing.T) {
	enabled := true
	config := &common.OdigosConfiguration{Interrogation: &common.InterrogationConfiguration{Enabled: &enabled}}
	apiKeyEntry := func(prov map[string]string) *string {
		result, err := EffectiveConfigToModel(config, prov)
		require.NoError(t, err)
		for _, entry := range result.Provenance {
			if entry.HelmPath == services.InterrogationAPIKeyProvenancePath {
				return &entry.ReconciledFrom
			}
		}
		return nil
	}

	require.Nil(t, apiKeyEntry(nil), "a Helm or missing credential is not attributed")
	got := apiKeyEntry(map[string]string{services.InterrogationAPIKeyProvenancePath: consts.OdigosLocalUiConfigName})
	require.NotNil(t, got)
	require.Equal(t, consts.OdigosLocalUiConfigName, *got)
}
