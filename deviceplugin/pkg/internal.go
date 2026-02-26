package pkg

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation"
)

// Start device manager
// the device manager library doesn't support passing a context,
// however, internally it uses a context to cancel the device manager once SIGTERM or SIGINT is received.
// We run it outside of the error group to avoid blocking on Wait() in case of a fatal error.
func runDeviceManager() error {
	logger := commonlogger.Logger()
	logger.Info("Starting device manager")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start debug server
	debugDone := make(chan struct{})
	go func() {
		defer close(debugDone)
		err := common.StartDebugServer(ctx, logger, int(k8sconsts.DevicePluginDebugPort))
		if err != nil {
			logger.Error("Failed to start debug server", "err", err)
		} else {
			logger.Info("Debug server exited")
		}
	}()

	lister, err := instrumentation.NewLister(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device manager lister: %w", err)
	}

	manager := dpm.NewManager(lister, commonlogger.FromSlogHandler())
	manager.Run(ctx)

	// Wait for debug server to finish
	<-debugDone
	return nil
}
