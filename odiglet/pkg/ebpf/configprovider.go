package ebpf

import (
	"context"
	"errors"
	"sync"
	"time"

	commonlogger "github.com/odigos-io/odigos/common/logger"
)

// Max time to wait for a config update to be sent to the instrumentation.
// In case the configuration is updating in a faster pace relative to the pace the instrumentation can apply it,
// this timeout will be hit and avoid sending new configuration before the previous one is applied.
// This timeout might be changed in the future.
const applyConfigTimeout = 50 * time.Millisecond

type ConfigProvider[C any] struct {
	configChan    chan C
	initialConfig C

	stoppedMutex sync.Mutex
	stopped      bool
}

// NewConfigProvider creates a new configProvider with the given initial config.
// It allows for updating the configuration of a running instrumentation.
func NewConfigProvider[C any](initialConfig C) *ConfigProvider[C] {
	return &ConfigProvider[C]{
		initialConfig: initialConfig,
		configChan:    make(chan C),
	}
}

func (c *ConfigProvider[C]) InitialConfig(_ context.Context) C {
	return c.initialConfig
}

func (c *ConfigProvider[C]) Shutdown(_ context.Context) error {
	c.stoppedMutex.Lock()
	defer c.stoppedMutex.Unlock()

	if c.stopped {
		return nil
	}

	close(c.configChan)
	c.stopped = true
	return nil
}

func (c *ConfigProvider[C]) Watch() <-chan C {
	return c.configChan
}

// SendConfig sends a new configuration to the instrumentation.
// If the instrumentation was closed or cannot accept the new configuration within a configured timeout,
// an error is returned.
func (c *ConfigProvider[C]) SendConfig(ctx context.Context, newConfig C) error {
	c.stoppedMutex.Lock()
	defer c.stoppedMutex.Unlock()

	if c.stopped {
		commonlogger.Logger().Info("SendConfig called on stopped configProvider, the supplied config will be ignored")
		return nil
	}

	// send a config or potentially return an error on timeout
	applyCtx, cancel := context.WithTimeout(ctx, applyConfigTimeout)
	defer cancel()

	select {
	case c.configChan <- newConfig:
		return nil
	case <-applyCtx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("failed to update config of instrumentation: timeout waiting for config update")
		}
		return ctx.Err()
	}
}
