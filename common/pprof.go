package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-logr/logr"
)

// StartPprofServer starts the pprof server on the specified port. This is blocking, so it should be run in a goroutine.
func StartPprofServer(ctx context.Context, logger logr.Logger, port int) error {
	logger.Info("Starting pprof server", "port", port)
	addr := fmt.Sprintf(":%d", port)

	server := &http.Server{Addr: addr, Handler: nil,
		ReadHeaderTimeout: time.Second * 5}
	done := make(chan struct{})
	errChan := make(chan error, 1)

	go func() {
		defer close(done)
		defer close(errChan)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error(err, "unable to start pprof server")
			errChan <- err
		}
	}()

	// Wait for server startup errors or context cancellation
	select {
	case err := <-errChan:
		if err != nil {
			return err // Return if there was an error starting the server
		}
	case <-ctx.Done():
	}

	// Shutdown the server if the context is done
	if err := server.Shutdown(ctx); err != nil {
		logger.Error(err, "error shutting down pprof server")
		return err
	}

	<-done
	return nil
}
