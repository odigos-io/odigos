package main

import (
	"os"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/odiglet"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "net/http/pprof"

	"github.com/odigos-io/odigos/common"
	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/env"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/instrumentlang"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	// Init Kubernetes clientset
	cfg, err := rest.InClusterConfig()
	if err != nil {
		panic(err)
	}

	// Increase the QPS and Burst to avoid client throttling
	// Observed mainly in large clusters once updating big amount of instrumentationInstances
	cfg.QPS = 200
	cfg.Burst = 200

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		panic(err)
	}

	// If started in init mode
	if len(os.Args) == 2 && os.Args[1] == "init" {
		odiglet.OdigletInitPhase(clientset)
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

	dg, err := distros.NewCommunityGetter()
	if err != nil {
		log.Logger.Error(err, "Failed to create distro getter")
		os.Exit(1)
	}

	instrumentationManagerOptions := ebpf.InstrumentationManagerOptions{
		Factories:          ebpfInstrumentationFactories(),
		DistributionGetter: dg,
	}

	o, err := odiglet.New(clientset, deviceInjectionCallbacks(), instrumentationManagerOptions)
	if err != nil {
		log.Logger.Error(err, "Failed to initialize odiglet")
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()
	o.Run(ctx)

	log.Logger.V(0).Info("odiglet exiting")
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

func ebpfInstrumentationFactories() map[commonInstrumentation.OtelDistribution]commonInstrumentation.Factory {
	return map[commonInstrumentation.OtelDistribution]commonInstrumentation.Factory{
		commonInstrumentation.OtelDistribution{
			Language: common.GoProgrammingLanguage,
			OtelSdk:  common.OtelSdkEbpfCommunity,
		}: sdks.NewGoInstrumentationFactory(),
	}
}
