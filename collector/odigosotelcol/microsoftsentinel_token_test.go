package main

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/policy"
	"github.com/Azure/azure-sdk-for-go/sdk/azidentity"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type sentinelTokenTransport struct{ tokens int }

func (s *sentinelTokenTransport) Do(req *http.Request) (*http.Response, error) {
	var body string
	switch {
	case strings.Contains(req.URL.Path, ".well-known/openid-configuration"):
		body = `{"authorization_endpoint":"https://login.microsoftonline.com/tenant/oauth2/v2.0/authorize","token_endpoint":"https://login.microsoftonline.com/tenant/oauth2/v2.0/token","issuer":"https://login.microsoftonline.com/tenant/v2.0"}`
	case strings.HasSuffix(req.URL.Path, "/token"):
		s.tokens++
		expires := 1
		if s.tokens > 1 {
			expires = 3600
		}
		body = fmt.Sprintf(`{"access_token":"test-token-%d","token_type":"Bearer","expires_in":%d,"scope":"https://monitor.azure.com/.default"}`, s.tokens, expires)
	default:
		return nil, fmt.Errorf("unexpected token request: %s", req.URL.Path)
	}
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(body)), Request: req}, nil
}

func TestMicrosoftSentinelEntraTokenRenewal(t *testing.T) {
	transport := &sentinelTokenTransport{}
	credential, err := azidentity.NewClientSecretCredential("tenant", "client", "secret", &azidentity.ClientSecretCredentialOptions{
		DisableInstanceDiscovery: true, ClientOptions: azcore.ClientOptions{Transport: transport},
	})
	require.NoError(t, err)
	request := policy.TokenRequestOptions{Scopes: []string{"https://monitor.azure.com/.default"}}
	first, err := credential.GetToken(context.Background(), request)
	require.NoError(t, err)
	renewed, err := credential.GetToken(context.Background(), request)
	require.NoError(t, err)
	assert.NotEqual(t, first.Token, renewed.Token)
	assert.Equal(t, 2, transport.tokens)
	cached, err := credential.GetToken(context.Background(), request)
	require.NoError(t, err)
	assert.Equal(t, renewed.Token, cached.Token)
	assert.Equal(t, 2, transport.tokens)
}
