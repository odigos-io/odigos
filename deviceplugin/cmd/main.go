package main

import (
	"os"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/deviceplugin/pkg"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation"
	"github.com/odigos-io/odigos/deviceplugin/pkg/log"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
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

	dp := pkg.New(pkg.Options{
		DeviceInjectionCallbacks: deviceInjectionCallbacks(),
	})

	if err := dp.Run(); err != nil {
		log.Logger.Error(err, "Device plugin exited with error")
	}
}

func deviceInjectionCallbacks() instrumentation.OtelSdksLsf {
	// still being used in enterprise.
	// will be removed once dotnet-legacy is removed and we stay with only "generic" device.
	return map[common.ProgrammingLanguage]map[common.OtelSdk]instrumentation.LangSpecificFunc{}
}
