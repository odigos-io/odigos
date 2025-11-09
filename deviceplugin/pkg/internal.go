package pkg

import (
	"context"
	"fmt"

	"github.com/odigos-io/odigos-device-plugin/pkg/dpm"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
)

// Start device manager
// the device manager library doesn't support passing a context,
// however, internally it uses a context to cancel the device manager once SIGTERM or SIGINT is received.
// We run it outside of the error group to avoid blocking on Wait() in case of a fatal error.
func runDeviceManager() error {
	log.Logger.V(0).Info("Starting device manager")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	lister, err := instrumentation.NewLister(ctx)
	if err != nil {
		return fmt.Errorf("failed to create device manager lister: %w", err)
	}

	manager := dpm.NewManager(lister, log.Logger)
	manager.Run(ctx)
	return nil
}
