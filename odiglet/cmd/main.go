package main

import (
	"flag"
	"os"

	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/odiglet"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "net/http/pprof"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL"))
	logger := commonlogger.Logger()

	var healthProbeBindPort int
	flag.IntVar(&healthProbeBindPort, "health-probe-bind-port", k8sconsts.OdigletDefaultHealthProbeBindPort, "The port the probe endpoint binds to.")
	flag.Parse()

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

	logger.Info("Starting odiglet")

	// Load env
	if err := env.Load(); err != nil {
		logger.Error("Failed to load env", "err", err)
		os.Exit(1)
	}

	dg, err := distros.NewCommunityGetter()
	if err != nil {
		logger.Error("Failed to create distro getter", "err", err)
		os.Exit(1)
	}

	instrumentationManagerOptions := ebpf.InstrumentationManagerOptions{
		Factories:                  ebpfInstrumentationFactories(),
		DistributionGetter:         dg,
		OdigletHealthProbeBindPort: healthProbeBindPort,
	}

	o, err := odiglet.New(clientset, instrumentationManagerOptions)
	if err != nil {
		logger.Error("Failed to initialize odiglet", "err", err)
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()
	o.Run(ctx)

	logger.Info("odiglet exiting")
}

func ebpfInstrumentationFactories() map[string]commonInstrumentation.Factory {
	return map[string]commonInstrumentation.Factory{
		"golang-community": sdks.NewGoInstrumentationFactory(),
	}
}
