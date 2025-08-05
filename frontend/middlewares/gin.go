package middlewares

import (
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/services"
)

func SecurityHeadersMiddleware(c *gin.Context) {
	c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
	c.Writer.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	c.Writer.Header().Set("X-Frame-Options", "DENY")
	c.Next()
}

func OidcMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		oauth2Config, err := services.GetOidcOauthConfig(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Error getting OIDC OAuth2 config: %s", err.Error())})
			return
		}

		// We're in a middleware, so we should check OIDC token only if OAuth2 is configured here
		if oauth2Config != nil {
			token, err := c.Cookie("id_token")

			// If no token is present, redirect to OIDC auth
			if token == "" {
				services.RedirectToOidcAuth(c, oauth2Config)
				return
			}

			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Error getting OIDC token from cookies: %s", err.Error())})
				return
			}

			oidcTokenVerifier, err := services.GetOidcTokenVerifier(ctx)
			if err != nil || oidcTokenVerifier == nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Error getting OIDC token verifier: %s", err.Error())})
				return
			}

			// Verify the OIDC token
			idToken, err := oidcTokenVerifier.Verify(ctx, token)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": fmt.Sprintf("Error verifiying OIDC token: %s", err.Error())})
				return
			}

			// Optionally set values into the context for handlers to use
			c.Set("uid", idToken.Subject)
		}

		c.Next()
	}
}
