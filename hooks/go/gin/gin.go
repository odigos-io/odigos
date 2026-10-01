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
			var recovered any
			go func(ctx context.Context) {
				// The middleware chain runs on this goroutine, so a panic in it
				// is out of reach of gin's Recovery middleware, which is deferred
				// on the request goroutine. Left unhandled it would take down the
				// whole process and leave the receive below blocked forever, so
				// carry the panic value back and re-raise it on the caller.
				defer func() {
					recovered = recover()
					done <- true
				}()
				middlewareName := runtime.FuncForPC(reflect.ValueOf(middlewareCopy).Pointer()).Name()
				executeMiddleware(ctx, c, middlewareName, middlewareCopy)
			}(reqCtx)
			<-done
			cancel()
			if recovered != nil {
				panic(recovered)
			}
		}
	}

	return wrappedMiddlewares
}

//go:noinline
func executeMiddleware(ctx context.Context, c *gin.Context, name string, next gin.HandlerFunc) {
	next(c)
}
