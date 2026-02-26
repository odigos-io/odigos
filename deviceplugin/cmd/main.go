package main

import (
	"os"

	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/deviceplugin/pkg"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	_ "net/http/pprof"
)

func main() {
	commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL"))
	logger := commonlogger.Logger()

	// Load env
	if err := env.Load(); err != nil {
		logger.Error("Failed to load env", "err", err)
		os.Exit(1)
	}

	dp := pkg.New()

	if err := dp.Run(); err != nil {
		logger.Error("Device plugin exited with error", "err", err)
	}
}
