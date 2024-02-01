package odigosresourcenameprocessor

import (
	"errors"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	ErrNoDeviceFound = errors.New("no device found")
)

type NameResolver struct {
	logger                      *zap.Logger
	kubelet                     *kubeletClient
	mu                          sync.RWMutex
	devicesToResourceAttributes map[string]*K8sResourceAttributes
	shutdown                    chan struct{}
	shutdownOnce                sync.Once
}

func (n *NameResolver) Resolve(deviceID string) (*K8sResourceAttributes, error) {
	n.mu.RLock()
	resourceAttributes, ok := n.devicesToResourceAttributes[deviceID]
	n.mu.RUnlock()

	if !ok {
		err := n.updateDevicesToPods()
		if err != nil {
			n.logger.Error("Error updating devices to pods", zap.Error(err))
			return &K8sResourceAttributes{}, err
		}

		n.mu.RLock()
		resourceAttributes, ok = n.devicesToResourceAttributes[deviceID]
		n.mu.RUnlock()

		if !ok {
			return &K8sResourceAttributes{}, ErrNoDeviceFound
		}
	}

	return resourceAttributes, nil
}

func (n *NameResolver) updateDevicesToPods() error {
	allocations, err := n.kubelet.GetAllocations()
	if err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.devicesToResourceAttributes = allocations
	return nil
}

func (n *NameResolver) Start() error {
	n.logger.Info("Starting NameResolver ...")
	go func() {
		// Refresh devices to pods every 5 seconds
		ticker := time.NewTicker(5 * time.Second)
		for {
			select {
			case <-ticker.C:
				err := n.updateDevicesToPods()
				if err != nil {
					n.logger.Error("Error updating devices to pods", zap.Error(err))
				}
			case <-n.shutdown:
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (n *NameResolver) Shutdown() {
	n.logger.Info("Shutting down NameResolver ...")
	n.shutdownOnce.Do(func() {
		close(n.shutdown)
	})
}
