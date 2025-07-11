package services

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"golang.org/x/oauth2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

// gets the OIDC configuration values from the odigos-config ConfigMap
func getOidcValuesFromConfig(ctx context.Context) (string, string, string, string, bool) {
	var odigosConfiguration common.OdigosConfiguration
	odigosns := env.GetCurrentNamespace()

	configMap, err := kube.DefaultClient.CoreV1().ConfigMaps(odigosns).Get(ctx, consts.OdigosConfigurationName, metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Error getting CM: %v\n", err)
	}
	err = yaml.Unmarshal([]byte(configMap.Data[consts.OdigosConfigurationFileName]), &odigosConfiguration)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v\n", err)
	}
	if odigosConfiguration.Oidc == nil {
		return "", "", "", "", false
	}

	// UI values
	uiRemoteUrl := odigosConfiguration.UiRemoteUrl
	if uiRemoteUrl == "" {
		uiRemoteUrl = "http://localhost:3000"
	}
	if !strings.HasSuffix(uiRemoteUrl, "/auth/callback") {
		uiRemoteUrl = fmt.Sprintf("%s/auth/callback", uiRemoteUrl)
	}

	// OIDC values
	secret, err := kube.DefaultClient.CoreV1().Secrets(odigosns).Get(ctx, consts.OidcSecretName, metav1.GetOptions{})
	if err != nil {
		return "", "", "", "", false
	}
	oidcClientSecret := string(secret.Data[consts.OidcClientSecretProperty])
	oidcClientId := odigosConfiguration.Oidc.ClientId
	oidcTenantUrl := odigosConfiguration.Oidc.TenantUrl
	if !strings.HasPrefix(oidcTenantUrl, "https://") {
		oidcTenantUrl = fmt.Sprintf("https://%s", oidcTenantUrl)
	}

	shouldProcessOidc := oidcTenantUrl != "" && oidcClientId != "" && oidcClientSecret != ""

	return uiRemoteUrl, oidcTenantUrl, oidcClientId, oidcClientSecret, shouldProcessOidc
}

func getOidcProvider(ctx context.Context) (string, string, string, string, *oidc.Provider, error) {
	uiRemoteUrl, oidcTenantUrl, oidcClientId, oidcClientSecret, shouldProcessOidc := getOidcValuesFromConfig(ctx)
	if !shouldProcessOidc {
		return "", "", "", "", nil, nil
	}

	// Create a Provider through discovery
	oidcProvider, err := oidc.NewProvider(ctx, oidcTenantUrl)
	if err != nil {
		return "", "", "", "", nil, fmt.Errorf("error initializing OIDC provider: %w", err)
	}

	return uiRemoteUrl, oidcTenantUrl, oidcClientId, oidcClientSecret, oidcProvider, nil
}

func GetOidcTokenVerifier(ctx context.Context) (*oidc.IDTokenVerifier, error) {
	_, _, oidcClientId, _, oidcProvider, err := getOidcProvider(ctx)
	if err != nil {
		return nil, err
	}
	if oidcProvider == nil {
		// no provder, this means OIDC is not configured
		return nil, nil
	}

	// Create a token verifier
	oidcTokenVerifier := oidcProvider.Verifier(&oidc.Config{
		ClientID: oidcClientId,
	})

	return oidcTokenVerifier, nil
}

func GetOidcOauthConfig(ctx context.Context) (*oauth2.Config, error) {
	uiRemoteUrl, _, oidcClientId, oidcClientSecret, oidcProvider, err := getOidcProvider(ctx)
	if err != nil {
		return nil, err
	}
	if oidcProvider == nil {
		// no provder, this means OIDC is not configured
		return nil, nil
	}

	// Configure OAuth2 with the provider's endpoints
	oauth2Config := &oauth2.Config{
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  uiRemoteUrl,
		ClientID:     oidcClientId,
		ClientSecret: oidcClientSecret,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return oauth2Config, nil
}

func RedirectToOidcAuth(c *gin.Context, oauth2Config *oauth2.Config) {
	state, err := randString(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	nonce, err := randString(16)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
		return
	}

	setCallbackCookie(c.Writer, c.Request, "state", state, "", 0)
	setCallbackCookie(c.Writer, c.Request, "nonce", nonce, "", 0)

	c.Redirect(http.StatusFound, oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)))
}

func OidcAuthCallback(ctx context.Context, c *gin.Context) {
	// Initialize OIDC & OAuth2
	oidcTokenVerifier, err := GetOidcTokenVerifier(ctx)
	if err != nil {
		log.Fatalf("Error initializing OIDC verifier: %s\n", err)
	}
	oauth2Config, err := GetOidcOauthConfig(ctx)
	if err != nil {
		log.Fatalf("Error initializing OAuth2 config: %s\n", err)
	}
	// We're in a callback (after being redirected from auth),
	// so we should always have OIDC & OAuth2 configured here.
	if oidcTokenVerifier == nil || oauth2Config == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "OIDC or OAuth2 is not configured"})
		return
	}

	urlQuery := c.Request.URL.Query()

	// Verify the 'state' from the query parameters against the cookies
	state, err := c.Cookie("state")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"state": err.Error()})
		return
	}
	if urlQuery.Get("state") != state {
		c.JSON(http.StatusInternalServerError, gin.H{"state": "mismatch"})
		return
	}

	// Exchange the authorization code for an OAuth2 token
	code := urlQuery.Get("code")
	oauth2Token, err := oauth2Config.Exchange(ctx, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"token exchange": err.Error()})
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"id_token": "missing in oauth2 token"})
		return
	}
	idToken, err := oidcTokenVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"token verify": err.Error()})
		return
	}

	// Verify the 'nonce' from the cookies against the token payload
	nonce, err := c.Cookie("nonce")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"nonce": err.Error()})
		return
	}
	if idToken.Nonce != nonce {
		c.JSON(http.StatusInternalServerError, gin.H{"nonce": "mismatch"})
		return
	}

	setCallbackCookie(c.Writer, c.Request, "id_token", rawIDToken, "/", int(idToken.Expiry.Unix()-idToken.IssuedAt.Unix()))

	c.Redirect(http.StatusFound, "/")
}
