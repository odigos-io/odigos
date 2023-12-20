package odigosresourcenameprocessor

import (
	"errors"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	"sync"
	"time"
)

const (
	nodeNameEnvVar = "NODE_NAME"
)

var (
	ErrNoDeviceFound = errors.New("no device found")
)

type NameResolver struct {
	kc            kubernetes.Interface
	logger        *zap.Logger
	kubelet       *kubeletClient
	mu            sync.RWMutex
	devicesToPods map[string]string
	shutdown      chan struct{}
}

func (n *NameResolver) Resolve(deviceID string) (string, error) {
	n.mu.RLock()
	name, ok := n.devicesToPods[deviceID]
	n.mu.RUnlock()

	if !ok {
		err := n.updateDevicesToPods()
		if err != nil {
			n.logger.Error("Error updating devices to pods", zap.Error(err))
			return "", err
		}

		n.mu.RLock()
		name, ok = n.devicesToPods[deviceID]
		n.mu.RUnlock()

		if !ok {
			return "", ErrNoDeviceFound
		}
	}

	return name, nil
}

func (n *NameResolver) updateDevicesToPods() error {
	allocations, err := n.kubelet.GetAllocations()
	if err != nil {
		return err
	}

	n.mu.Lock()
	defer n.mu.Unlock()

	n.devicesToPods = allocations
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
	close(n.shutdown)
}
