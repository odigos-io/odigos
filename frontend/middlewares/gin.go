package middlewares

import (
	"context"
	"log"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/frontend/services"
)

func SecurityHeadersMiddleware(c *gin.Context) {
	// Skip CSP for static files and playground to avoid blocking JavaScript resources
	if !isStaticFile(c.Request.URL.Path) && c.Request.URL.Path != "/playground" {
		// connect-src allows the public Go offsets bucket: the browser downloads the
		// manifest itself (the UI pod may have no outbound internet). @odigos/ui-kit
		// fetches it from GO_OFFSETS_PUBLIC_URL, which must stay in this bucket.
		c.Writer.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline' fonts.googleapis.com; font-src 'self' fonts.googleapis.com fonts.gstatic.com; img-src 'self' data: blob:; connect-src 'self' ws: wss: "+consts.GoOffsetsPublicBucketURL+";")
	}
	c.Writer.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
	c.Writer.Header().Set("X-Frame-Options", "DENY")
	c.Next()
}

// isStaticFile checks if the request is for a static file
func isStaticFile(path string) bool {
	staticExtensions := []string{".js", ".css", ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".woff", ".woff2", ".ttf", ".eot"}
	for _, ext := range staticExtensions {
		if len(path) >= len(ext) && path[len(path)-len(ext):] == ext {
			return true
		}
	}
	// Also check for Next.js static paths
	if len(path) >= 7 && path[:7] == "/_next/" {
		return true
	}
	return false
}

func OidcMiddleware(ctx context.Context) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestCtx, cancel := services.OidcRequestContext(ctx, c.Request.Context())
		defer cancel()
		oauth2Config, verifier, err := services.GetOidcAuthConfig(requestCtx)
		if err != nil {
			log.Printf("OIDC authentication configuration unavailable: %v", err)
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"message": "OIDC configuration is unavailable"})
			return
		}
		if oauth2Config == nil {
			c.Next()
			return
		}

		var token string
		if authorization := c.GetHeader("Authorization"); authorization != "" {
			parts := strings.Fields(authorization)
			if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid bearer token"})
				return
			}
			token = parts[1]
		} else {
			token, _ = c.Cookie("id_token")
		}
		if token == "" {
			if c.FullPath() == "" {
				services.RedirectToOidcAuth(c, oauth2Config)
				c.Abort()
			} else {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Authentication required"})
			}
			return
		}

		idToken, err := verifier.Verify(requestCtx, token)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"message": "Invalid or expired OIDC token"})
			return
		}
		c.Set("uid", idToken.Subject)
		c.Next()
	}
}
