package common

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// registerOnce ensures /debug/loglevel is registered on http.DefaultServeMux
// exactly once per process, even if StartDebugServer is called multiple times
// (e.g. in tests with -count>1).
var registerOnce sync.Once

// logLevelHandler handles GET and PUT /debug/loglevel requests.
//
// GET  /debug/loglevel         → returns current level as plain text (e.g. "info")
// PUT  /debug/loglevel?level=debug → changes the level; returns the new level
func logLevelHandler(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		_, _ = fmt.Fprintln(w, commonlogger.CurrentLevel())
	case http.MethodPut:
		lvl := r.URL.Query().Get("level")
		if lvl == "" {
			http.Error(w, "missing ?level= query parameter", http.StatusBadRequest)
			return
		}
		if err := commonlogger.SetLevel(lvl); err != nil {
			http.Error(w, fmt.Sprintf("invalid log level %q (accepted: debug, info, warn, error)", lvl), http.StatusBadRequest)
			return
		}
		_, _ = fmt.Fprintln(w, commonlogger.CurrentLevel())
	default:
		http.Error(w, "method not allowed; use GET or PUT", http.StatusMethodNotAllowed)
	}
}

// StartDebugServer starts the debug/admin HTTP server on the specified port.
// It serves the Go pprof suite (/debug/pprof/*) and the log-level control
// endpoint (/debug/loglevel). This is blocking, so it should be run in a goroutine.
func StartDebugServer(ctx context.Context, logger *slog.Logger, port int) error {
	registerOnce.Do(func() {
		http.HandleFunc("/debug/loglevel", logLevelHandler)
	})
	logger.Info("Starting debug server", "port", port)
	addr := fmt.Sprintf(":%d", port)

	server := &http.Server{Addr: addr, Handler: nil,
		ReadHeaderTimeout: time.Second * 5}
	done := make(chan struct{})
	errChan := make(chan error, 1)

	go func() {
		defer close(done)
		defer close(errChan)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("unable to start debug server", "err", err)
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
		logger.Error("error shutting down debug server", "err", err)
		return err
	}

	<-done
	return nil
}
