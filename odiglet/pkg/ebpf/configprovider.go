package ebpf

import (
	"context"
	"errors"
	"sync"

	"github.com/odigos-io/odigos/odiglet/pkg/log"
)

type configProvider[C any] struct {
	configChan    chan C
	initialConfig C

	stoppedMutex sync.Mutex
	stopped      bool
}

// NewConfigProvider creates a new configProvider with the given initial config.
// It allows for updating the configuration of a running instrumentation.
func NewConfigProvider[C any](initialConfig C) *configProvider[C] {
	return &configProvider[C]{
		initialConfig: initialConfig,
		configChan:    make(chan C),
	}
}

func (c *configProvider[C]) InitialConfig(_ context.Context) C {
	return c.initialConfig
}

func (c *configProvider[C]) Shutdown(_ context.Context) error {
	c.stoppedMutex.Lock()
	defer c.stoppedMutex.Unlock()

	if c.stopped {
		return nil
	}

	close(c.configChan)
	c.stopped = true
	return nil
}

func (c *configProvider[C]) Watch() <-chan C {
	return c.configChan
}

func (c *configProvider[C]) SendConfig(ctx context.Context, newConfig C) error {
	c.stoppedMutex.Lock()
	defer c.stoppedMutex.Unlock()

	if c.stopped {
		log.Logger.Info("SendConfig called on stopped configProvider, the supplied config will be ignored")
		return nil
	}

	// send a config or potentially return an error on timeout
	// TODO: we should decide on a timeout value for sending config.
	select {
	case c.configChan <- newConfig:
		return nil
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			return errors.New("failed to update config of instrumentation: timeout waiting for config update")
		}
		return ctx.Err()
	}
}
