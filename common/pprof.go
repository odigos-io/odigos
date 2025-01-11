package common

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-logr/logr"

	"github.com/odigos-io/odigos/common/consts"
)

// StartPprofServer starts the pprof server. This is blocking, so it should be run in a goroutine.
func StartPprofServer(ctx context.Context, logger logr.Logger) error {
	logger.Info("Starting pprof server")
	addr := fmt.Sprintf(":%d", consts.PprofOdigosPort)

	server := &http.Server{Addr: addr, Handler: nil}
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
