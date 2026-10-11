package server

import (
	"context"
	"fmt"
	"os"
	"sync"

	"github.com/gin-gonic/gin"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// ServeAndWait runs the gin server in a goroutine, blocks on sigCh until a
// signal arrives, then cancels the supplied context and waits for the
// background goroutines (started by StartBackground) to drain. Returns an
// error only if the listener fails to start.
func ServeAndWait(
	cancel context.CancelFunc,
	deps *Deps,
	r *gin.Engine,
	sigCh <-chan os.Signal,
	wg *sync.WaitGroup,
) error {
	log := commonlogger.LoggerCompat().With("subsystem", "startup")
	addr := fmt.Sprintf("%s:%d", deps.Flags.Address, deps.Flags.Port)
	url := fmt.Sprintf("http://%s", addr)

	listenErr := make(chan error, 1)
	go func() {
		log.Info("Odigos UI is available", "address", url)
		if err := r.Run(addr); err != nil {
			listenErr <- err
		}
	}()

	select {
	case <-sigCh:
		log.Info("Shutting down Odigos UI...")
	case err := <-listenErr:
		log.Error("listener failed", "err", err)
		cancel()
		if wg != nil {
			wg.Wait()
		}
		return err
	}

	cancel()
	if wg != nil {
		wg.Wait()
	}
	return nil
}
