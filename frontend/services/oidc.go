package services

import (
	"context"
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
