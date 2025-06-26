package gin

import (
	"context"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
)

// OdigosGinMiddlewareHandler accepts a list of gin.HandlerFuncs and returns that list
// with each function wrapped in a new function that calls executeMiddleware.
//
//go:noinline
func OdigosGinMiddleware(middlewares ...gin.HandlerFunc) []gin.HandlerFunc {
	wrappedMiddlewares := make([]gin.HandlerFunc, len(middlewares))

	for i, middleware := range middlewares {
		// Create a local copy to avoid closure bug
		middlewareCopy := middleware
		wrappedMiddlewares[i] = func(c *gin.Context) {
			done := make(chan bool)
			reqCtx, cancel := context.WithCancel(c.Request.Context())
			go func(ctx context.Context) {
				middlewareName := runtime.FuncForPC(reflect.ValueOf(middlewareCopy).Pointer()).Name()
				executeMiddleware(ctx, c, middlewareName, middlewareCopy)
				done <- true
			}(reqCtx)
			<-done
			cancel()
		}
	}

	return wrappedMiddlewares
}

//go:noinline
func executeMiddleware(ctx context.Context, c *gin.Context, name string, next gin.HandlerFunc) {
	next(c)
}
