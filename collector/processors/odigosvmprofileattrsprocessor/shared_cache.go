package odigosvmprofileattrsprocessor

import (
	"context"
	"sync"

	"go.uber.org/zap"

	"github.com/odigos-io/odigos/common/unixfd"
)

// The PID→service.name cache and its single unixfd client are process-global, not
// per-processor. This is deliberate: a collector config reload (SIGHUP) does a full
// service.Shutdown()+rebuild, which destroys and recreates the processor. If the
// cache lived on the processor, every reload would drop it, force a reconnect+reset,
// and the rebuilt processor would enrich from an empty cache during warm-up —
// silently dropping profiles on every config change (add a source/destination, etc.).
//
// By owning the cache + client at package scope on a context.Background()-scoped
// goroutine, the connection stays open and the cache stays warm across reloads. The
// rebuilt processor simply references the same surviving cache. The singleton is
// created lazily on the first processor start() and lives for the whole process; it
// is intentionally never torn down on processor shutdown.
var (
	sharedOnce   sync.Once
	sharedCache  *profileAttrCache
	sharedCancel context.CancelFunc // cancels the client ctx; only used by tests for cleanup
	sharedDone   chan struct{}      // closed when the client goroutine exits (tests wait on it)
)

// sharedProfileAttrCache returns the process-global PID→attrs cache, starting the
// single unixfd client that maintains it on first call. socketPath/logger from the
// first caller win (all profiles processors in one collector share one VM-agent
// socket, so this is the same value for every instance).
func sharedProfileAttrCache(socketPath string, logger *zap.Logger) *profileAttrCache {
	sharedOnce.Do(func() {
		sharedCache = newProfileAttrCache()
		// Derived from context.Background(): the client must outlive any individual
		// processor instance so the cache survives config reloads. In production the
		// cancel is never called (lives until process exit); tests cancel it to clean
		// up the goroutine (goleak).
		ctx, cancel := context.WithCancel(context.Background())
		sharedCancel = cancel
		sharedDone = make(chan struct{})
		go func() {
			defer close(sharedDone)
			err := unixfd.ConnectAndListenProfileAttrs(ctx, socketPath, logger,
				func(line string) { sharedCache.applyEvent(line) },
				func() { sharedCache.reset() }, // wipe on each new session before snapshot replay
			)
			if err != nil && ctx.Err() == nil {
				logger.Error("shared profiles attr unix client stopped", zap.Error(err))
			}
		}()
	})
	return sharedCache
}
