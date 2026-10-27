package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/odigos-io/odigos/frontend/services"
)

// WithAdminOverrideFromRequest marks the request context with the admin override
// when the request carries the override header.
func WithAdminOverrideFromRequest(c *gin.Context) {
	if c.GetHeader(AdminOverrideHeader) == "true" {
		c.Request = c.Request.WithContext(WithAdminOverride(c.Request.Context()))
	}
}

// ReadonlyGuardMiddleware rejects state-changing REST requests in read-only mode,
// the REST counterpart of the GraphQL mutation check in OperationInterceptor.
func ReadonlyGuardMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		WithAdminOverrideFromRequest(c)
		ctx := c.Request.Context()
		if services.IsReadonlyMode(ctx) && !AdminOverrideFromContext(ctx) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"message": services.ErrorIsReadonly.Error()})
			return
		}
		c.Next()
	}
}
