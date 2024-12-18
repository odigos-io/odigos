package main

import (
	"os"

	"github.com/odigos-io/odigos/odiglet"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/fs"

	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/instrumentlang"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"

	_ "net/http/pprof"
)

func odigletInitPhase() {
	if err := log.Init(); err != nil {
		panic(err)
	}
	err := fs.CopyAgentsDirectoryToHost()
	if err != nil {
		log.Logger.Error(err, "Failed to copy agents directory to host")
		os.Exit(-1)
	}
	os.Exit(0)
}

func main() {
	// If started in init mode
	if len(os.Args) == 2 && os.Args[1] == "init" {
		odigletInitPhase()
	}

	if err := log.Init(); err != nil {
		panic(err)
	}

	log.Logger.V(0).Info("Starting odiglet")

	// Load env
	if err := env.Load(); err != nil {
		log.Logger.Error(err, "Failed to load env")
		os.Exit(1)
	}

	o, err := odiglet.New(otelSdkLsf(), factories())
	if err != nil {
		log.Logger.Error(err, "Failed to initialize odiglet")
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()
	o.Run(ctx)

	log.Logger.V(0).Info("odiglet exiting")
}

func otelSdkLsf() instrumentation.OtelSdksLsf{
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

func factories() map[commonInstrumentation.OtelDistribution]commonInstrumentation.Factory {
	return map[commonInstrumentation.OtelDistribution]commonInstrumentation.Factory{
		commonInstrumentation.OtelDistribution{
			Language: common.GoProgrammingLanguage,
			OtelSdk:  common.OtelSdkEbpfCommunity,
		}: sdks.NewGoInstrumentationFactory(),
	}
}
