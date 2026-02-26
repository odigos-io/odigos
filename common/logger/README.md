# Odigos Logger

Structured logger (slog) shared across Odigos components. Single global instance, optional context carry, and runtime level control.

## Setup

Call `Init` once at the start of `main()`:

```go
import commonlogger "github.com/odigos-io/odigos/common/logger"

func main() {
    commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL")) // "debug", "info", "warn", "error"
    // ...
}
```

## Usage

**Direct logging** — use the global logger anywhere in the same process:

```go
logger := commonlogger.Logger()
logger.Info("started", "component", "autoscaler")
logger.Debug("detail", "key", value)
logger.Error("failed", "err", err)
```

**With attributes** — attach key/value pairs to a sub-logger:

```go
logger := commonlogger.Logger().With("controller", "NodeCollector")
logger.Info("reconciling", "name", name)
```

## Controller-runtime (logr)

For components using controller-runtime, set the global logr logger so all controllers use the same backend:

```go
ctrl.SetLogger(commonlogger.FromSlogHandler())
```

Then `mgr.GetLogger()` and reconciler loggers write through this logger; level and format are shared.

## Context (request-scoped logger)

Store a logger in context for downstream code; resolve with fallback to global, then default:

```go
// Store (e.g. in HTTP middleware or request handler)
ctx = commonlogger.IntoContext(ctx, commonlogger.Logger().With("requestID", id))

// Retrieve later
logger := commonlogger.FromContext(ctx)
logger.Info("processing request")
```

If no logger is in context, `FromContext` returns the global logger (or `slog.Default()` if `Init` was never called).

