package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/odiglet"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf/sdks/obi"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	_ "net/http/pprof"

	"github.com/odigos-io/odigos/api/k8sconsts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

func main() {
	commonlogger.Init(os.Getenv("ODIGOS_LOG_LEVEL"), "odiglet")
	klog.SetLogger(commonlogger.ToLogr())
	logger := commonlogger.LoggerCompat().With("subsystem", "startup")

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

	ctx := signals.SetupSignalHandler()

	// create a single gRPC connection that will be shared across instrumentations
	// of different processes and different languages.
	// this allows us to keep a single connection to the local collector.
	// we share the underlying connection transport, but each instrumentation can have
	// its own exporter with its own lifecycle.
	// using WithGRPCConn when creating the exporter allows us to manage the lifecycle of this connection
	// which is open the entire lifecycle of the odiglet.
	otlpCommonConn, err := grpc.NewClient(
		fmt.Sprintf("localhost:%d", consts.OTLPPort),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		logger.Error("failed to create a gRPC connection to local collector", "error", err)
		os.Exit(1)
	}
	defer func() {
		err := otlpCommonConn.Close()
		if err != nil {
			logger.Error("failed to close common OTLP connection", "error", err)
		}
	}()

	obiManager := obi.NewManager()

	instrumentationManagerOptions := ebpf.InstrumentationManagerOptions{
		DistributionGetter:         dg,
		OdigletHealthProbeBindPort: healthProbeBindPort,
		OBIManager:                 obiManager,
	}

	factories, err := ebpfInstrumentationFactories(otlpCommonConn, obiManager)
	if err != nil {
		logger.Error("failed to create ebpf factories: %w", err)
		os.Exit(1)
	}

	instrumentationManagerOptions.Factories = factories

	o, err := odiglet.New(clientset, instrumentationManagerOptions)
	if err != nil {
		logger.Error("Failed to initialize odiglet", "err", err)
		os.Exit(1)
	}
	o.Run(ctx)

	logger.Info("odiglet exiting")
}

func ebpfInstrumentationFactories(otlpCommon *grpc.ClientConn, obiManager *obi.Manager) (map[string]commonInstrumentation.Factory, error) {
	goFactory, err := sdks.NewGoInstrumentationFactory(otlpCommon)
	if err != nil {
		return nil, fmt.Errorf("failed to create go instrumentation factory: %w", err)
	}
	return map[string]commonInstrumentation.Factory{
		"golang-community": goFactory,
		obi.DistroName:     obiManager,
	}, nil
}
