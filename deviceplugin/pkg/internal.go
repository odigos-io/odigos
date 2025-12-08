package pkg

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
)

// Start device manager with proper signal handling for graceful shutdown
func runDeviceManager() error {
	log.Logger.V(0).Info("Starting device manager")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		sig := <-sigChan
		log.Logger.V(0).Info("Received signal, initiating graceful shutdown", "signal", sig)
		cancel()
	}()

	// Start pprof server
	pprofDone := make(chan struct{})
	go func() {
		defer close(pprofDone)
		err := common.StartPprofServer(ctx, log.Logger, int(k8sconsts.DevicePluginPprofEndpointPort))
		if err != nil {
			log.Logger.Error(err, "Failed to start pprof server")
		} else {
			log.Logger.V(0).Info("Pprof server exited")
		}
	}()

	lister, err := instrumentation.NewLister(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device manager lister: %w", err)
	}

	manager := dpm.NewManager(lister, log.Logger)
	manager.Run(ctx)

	// Wait for pprof server to finish
	<-pprofDone
	return nil
}
