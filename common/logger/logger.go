package logger

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/go-logr/logr"
)

var (
	mu       sync.RWMutex
	levelVar = new(slog.LevelVar)
	instance *slog.Logger
	handler  slog.Handler
)

// Init sets up the global structured logger backed by slog.
// level should be one of: "debug", "info", "warn", "error".
// If level is empty or invalid, it defaults to INFO (same as when ODIGOS_LOG_LEVEL is unset).
//
// Init must be called once at the start of main() before any logging.
func Init(level string) {
	parsed, err := parseLevel(level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "odigos logger: %v, defaulting to info\n", err)
		parsed = slog.LevelInfo
	}
	levelVar.Set(parsed)

	h := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     levelVar,
		AddSource: true,
	})

	l := slog.New(h)

	mu.Lock()
	handler = h
	instance = l
	slog.SetDefault(l)
	mu.Unlock()
}

// Logger returns the global *slog.Logger. If Init has not been called yet,
// it returns slog.Default() so callers never receive a nil logger.
func Logger() *slog.Logger {
	mu.RLock()
	defer mu.RUnlock()
	if instance != nil {
		return instance
	}
	return slog.Default()
}

// Handler returns the slog.Handler used by the global logger.
// If Init has not been called yet, returns slog.Default().Handler().
// Use this to bridge to controller-runtime's logr-based logger:
//
//	ctrl.SetLogger(logr.FromSlogHandler(logger.Handler()))
func Handler() slog.Handler {
	mu.RLock()
	defer mu.RUnlock()
	if handler != nil {
		return handler
	}
	return slog.Default().Handler()
}

// FromSlogHandler returns a logr.Logger backed by the global slog handler.
// This is a convenience wrapper for controller-runtime components that
// require a logr.Logger.
func FromSlogHandler() logr.Logger {
	return logr.FromSlogHandler(Handler())
}

// SetLevel changes the global log level at runtime without restarting.
// Accepts: "debug", "info", "warn", "error".
// Returns an error if the provided level is invalid.
func SetLevel(level string) error {
	parsed, err := parseLevel(level)
	if err != nil {
		return err
	}
	levelVar.Set(parsed)
	Logger().Info("log level updated", "level", parsed.String())
	return nil
}

// CurrentLevel returns the active log level as a lowercase string: "debug", "info", "warn", or "error".
// Useful for exposing the current level via a diagnostics endpoint.
func CurrentLevel() string {
	switch levelVar.Level() {
	case slog.LevelDebug:
		return "debug"
	case slog.LevelWarn:
		return "warn"
	case slog.LevelError:
		return "error"
	default:
		return "info"
	}
}

// parseLevel converts a string to slog.Level.
func parseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return slog.LevelInfo, fmt.Errorf("unknown log level %q", s)
	}
}
