package main

import (
	"os"

	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/deviceplugin/pkg"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation"
	"github.com/odigos-io/odigos/deviceplugin/pkg/instrumentation/instrumentlang"
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
	return map[common.ProgrammingLanguage]map[common.OtelSdk]instrumentation.LangSpecificFunc{
		common.GoProgrammingLanguage: {
			common.OtelSdkEbpfCommunity: instrumentlang.Go,
		},
		common.JavaProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.Java,
		},
		common.PythonProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.Python,
		},
		common.JavascriptProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.NodeJS,
		},
		common.DotNetProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.DotNet,
		},
		common.NginxProgrammingLanguage: {
			common.OtelSdkNativeCommunity: instrumentlang.Nginx,
		},
	}
}
