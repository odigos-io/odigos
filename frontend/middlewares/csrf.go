package middlewares

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/services"
)

// CSRFMiddleware provides CSRF protection for state-changing operations
func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		csrfService := services.GetCSRFService()

		if !strings.HasPrefix(c.Request.URL.Path, "/graphql") {
			c.Next()
			return
		}

		_, err := c.Request.Cookie("csrf_token")
		if err == http.ErrNoCookie {
			c.Next()
			return
		}

		if err := csrfService.ValidateRequest(c.Request); err != nil {
			switch err {
			case services.ErrCSRFTokenMissing:
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "CSRF token missing",
					"code":  "CSRF_TOKEN_MISSING",
				})
			case services.ErrCSRFTokenInvalid:
				c.JSON(http.StatusForbidden, gin.H{
					"error": "CSRF token invalid",
					"code":  "CSRF_TOKEN_INVALID",
				})
			case services.ErrCSRFTokenExpired:
				c.JSON(http.StatusForbidden, gin.H{
					"error": "CSRF token expired",
					"code":  "CSRF_TOKEN_EXPIRED",
				})
			default:
				c.JSON(http.StatusForbidden, gin.H{
					"error": "CSRF validation failed",
					"code":  "CSRF_VALIDATION_FAILED",
				})
			}
			c.Abort()
			return
		}

		c.Next()
	}
}

// CSRFTokenHandler provides an endpoint to get CSRF tokens
func CSRFTokenHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		csrfService := services.GetCSRFService()
		csrfCookie := csrfService.GetCSRFToken(c.Request)

		var token string
		var err error

		if csrfCookie != "" && csrfService.ValidateToken(csrfCookie) == nil {
			token = csrfCookie
		} else {
			token, err = csrfService.GenerateToken()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to generate CSRF token",
				})
				return
			}
		}

		csrfService.SetCSRFCookie(c.Writer, c.Request, token)

		c.JSON(http.StatusOK, gin.H{
			"csrf_token": token,
		})
	}
}
