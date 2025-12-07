package main

import (
	"os"

	"github.com/odigos-io/odigos/deviceplugin/pkg"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	_ "net/http/pprof"
)

func main() {
	if err := log.Init(); err != nil {
		panic(err)
	}

	// Load env
	if err := env.Load(); err != nil {
		log.Logger.Error(err, "Failed to load env")
		os.Exit(1)
	}

	dp := pkg.New()

	if err := dp.Run(); err != nil {
		log.Logger.Error(err, "Device plugin exited with error")
	}
}
