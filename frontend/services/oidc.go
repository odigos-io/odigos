package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
)

func InitOidc(ctx context.Context) (*oidc.Provider, *oidc.IDTokenVerifier, *oauth2.Config, error) {
	// TODO: remove hardcoded values and use Odigos Config.
	oidcProviderUrl := "https://accounts.google.com"
	oidcRedirectUrl := ""
	oidcClientId := ""
	oidcClientSecret := ""

	// Create a Provider through discovery
	oidcProvider, err := oidc.NewProvider(ctx, oidcProviderUrl)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to create OIDC provider: %w", err)
	}

	// Configure OIDC
	oidcConfig := oidc.Config{
		ClientID: oidcClientId,
	}

	// Create a token verifier
	oidcVerifier := oidcProvider.Verifier(&oidcConfig)

	// Configure OAuth2 with the provider's endpoints
	oauth2Config := oauth2.Config{
		Endpoint:     oidcProvider.Endpoint(),
		RedirectURL:  oidcRedirectUrl,
		ClientID:     oidcClientId,
		ClientSecret: oidcClientSecret,
		Scopes:       []string{oidc.ScopeOpenID, "profile", "email"},
	}

	return oidcProvider, oidcVerifier, &oauth2Config, nil
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

	setCallbackCookie(c.Writer, c.Request, "state", state)
	setCallbackCookie(c.Writer, c.Request, "nonce", nonce)

	c.Redirect(http.StatusFound, oauth2Config.AuthCodeURL(state, oidc.Nonce(nonce)))
}

func OidcAuthCallback(ctx context.Context, c *gin.Context, oauth2Config *oauth2.Config) {
	urlQuery := c.Request.URL.Query()

	state, err := c.Cookie("state")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"cookie 'state' not found": err.Error()})
		return
	}
	if urlQuery.Get("state") != state.Value {
		c.JSON(http.StatusInternalServerError, gin.H{"query & cookie 'state' did not match": err.Error()})
		return
	}

	nonce, err := c.Cookie("nonce")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"cookie 'nonce' not found": err.Error()})
		return
	}
	if urlQuery.Get("code") != nonce.Value {
		c.JSON(http.StatusInternalServerError, gin.H{"query & cookie 'nonce' did not match": err.Error()})
		return
	}

	oauth2Token, err := oauth2Config.Exchange(ctx, nonce.Value)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"failed to exchange token": err.Error()})
		return
	}
	rawIDToken, ok := oauth2Token.Extra("id_token").(string)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"no 'id_token' field in oauth2 token": err.Error()})
		return
	}
	idToken, err := oidcVerifier.Verify(ctx, rawIDToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"failed to verify ID token": err.Error()})
		return
	}
	if idToken.Nonce != nonce.Value {
		c.JSON(http.StatusInternalServerError, gin.H{"token 'nonce' did not match": err.Error()})
		return
	}

	oauth2Token.AccessToken = "*REDACTED*"

	resp := struct {
		OAuth2Token   *oauth2.Token
		IDTokenClaims *json.RawMessage // ID Token payload is just JSON.
	}{oauth2Token, new(json.RawMessage)}

	if err := idToken.Claims(&resp.IDTokenClaims); err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	data, err := json.MarshalIndent(resp, "", "    ")
	if err != nil {
		http.Error(c.Writer, err.Error(), http.StatusInternalServerError)
		return
	}
	c.Writer.Write(data)
}
