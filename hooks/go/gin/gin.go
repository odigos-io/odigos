package gin

import (
	"context"
	"reflect"
	"runtime"

	"github.com/gin-gonic/gin"
)

// OdigosGinMiddleware is a middleware that executes a chain of Gin middlewares.
//
//go:noinline
func OdigosGinMiddleware(middlewares ...gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if len(middlewares) == 0 {
			c.Next()
			return
		}

		executeMiddlewareChain(c, middlewares)
	}
}

//go:noinline
func executeMiddlewareChain(c *gin.Context, middlewareChain []gin.HandlerFunc) {
	for i := 0; i < len(middlewareChain); i++ {
		if c.IsAborted() {
			return
		}

		done := make(chan bool)
		reqCtx, cancel := context.WithCancel(c.Request.Context())
		go func(ctx context.Context) {
			middlewareName := runtime.FuncForPC(reflect.ValueOf(middlewareChain[i]).Pointer()).Name()
			executeMiddleware(ctx, c, middlewareName, middlewareChain[i])
			done <- true
		}(reqCtx)
		<-done
		cancel()
	}
}

//go:noinline
func executeMiddleware(ctx context.Context, c *gin.Context, name string, next gin.HandlerFunc) {
	next(c)
}
