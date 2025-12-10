package odiglet

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/distros/distro"
	commonInstrumentation "github.com/odigos-io/odigos/instrumentation"
	criwrapper "github.com/odigos-io/odigos/k8sutils/pkg/cri"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"github.com/odigos-io/odigos/k8sutils/pkg/metrics"
	k8snode "github.com/odigos-io/odigos/k8sutils/pkg/node"
	"github.com/odigos-io/odigos/odiglet/pkg/ebpf"
	"github.com/odigos-io/odigos/odiglet/pkg/instrumentation/fs"
	"github.com/odigos-io/odigos/odiglet/pkg/kube"
	"github.com/odigos-io/odigos/odiglet/pkg/log"
	"github.com/odigos-io/odigos/odiglet/pkg/nodedetails"
	"github.com/odigos-io/odigos/opampserver/pkg/server"
	"golang.org/x/sync/errgroup"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Odiglet struct {
	clientset     *kubernetes.Clientset
	mgr           controllerruntime.Manager
	ebpfManager   commonInstrumentation.Manager
	configUpdates chan<- commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	criClient     *criwrapper.CriClient
}

const (
	configUpdatesBufferSize = 10
)

// New creates a new Odiglet instance.
func New(clientset *kubernetes.Clientset, instrumentationMgrOpts ebpf.InstrumentationManagerOptions) (*Odiglet, error) {
	err := feature.Setup()
	if err != nil {
		return nil, err
	}

	mgr, err := kube.CreateManager(instrumentationMgrOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to create controller-runtime manager %w", err)
	}

	// Create an OpenTelemetry MeterProvider that is based on controller-runtime prometheus registry
	// and register it as the global MeterProvider for the Odiglet
	provider, err := metrics.NewMeterProviderForController(resource.NewWithAttributes(
		semconv.SchemaURL,
		semconv.K8SNodeName(env.Current.NodeName),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create OpenTelemetry MeterProvider: %w", err)
	}
	otel.SetMeterProvider(provider)

	appendEnvVarNames := distro.GetAppendEnvVarNames(instrumentationMgrOpts.DistributionGetter.GetAllDistros())

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up ready check: %w", err)
	}

	configUpdates := make(chan commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup], configUpdatesBufferSize)
	ebpfManager, err := ebpf.NewManager(mgr.GetClient(), log.Logger, instrumentationMgrOpts, configUpdates, appendEnvVarNames)
	if err != nil {
		return nil, fmt.Errorf("failed to create ebpf manager %w", err)
	}
	criWrapper := criwrapper.CriClient{Logger: log.Logger}

	kubeManagerOptions := kube.KubeManagerOptions{
		Mgr:               mgr,
		ConfigUpdates:     configUpdates,
		CriClient:         &criWrapper,
		AppendEnvVarNames: appendEnvVarNames,
	}

	err = kube.SetupWithManager(kubeManagerOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to setup controller-runtime manager %w", err)
	}

	return &Odiglet{
		clientset:     clientset,
		mgr:           mgr,
		ebpfManager:   ebpfManager,
		configUpdates: configUpdates,
		criClient:     &criWrapper,
	}, nil
}

// Run starts the Odiglet components and blocks until the context is cancelled, or a critical error occurs.
func (o *Odiglet) Run(ctx context.Context) {
	g, groupCtx := errgroup.WithContext(ctx)

	if err := o.criClient.Connect(ctx); err != nil {
		log.Logger.Error(err, "Failed to connect to CRI runtime")
	}

	defer o.criClient.Close()

	// Start pprof server
	g.Go(func() error {
		err := common.StartPprofServer(groupCtx, log.Logger, int(k8sconsts.DefaultPprofEndpointPort))
		if err != nil {
			log.Logger.Error(err, "Failed to start pprof server")
		} else {
			log.Logger.V(0).Info("Pprof server exited")
		}
		// if we fail to start the pprof server, don't return an error as it is not critical
		// and we can run the rest of the components
		return nil
	})

	g.Go(func() error {
		err := o.ebpfManager.Run(groupCtx)
		if err != nil {
			log.Logger.Error(err, "Failed to run ebpf manager")
		}
		log.Logger.V(0).Info("eBPF manager exited")
		return err
	})

	// start OpAmp server
	odigosNs := env.GetCurrentNamespace()
	g.Go(func() error {
		err := server.StartOpAmpServer(groupCtx, log.Logger, o.mgr, o.clientset, env.Current.NodeName, odigosNs)
		if err != nil {
			log.Logger.Error(err, "Failed to start opamp server")
		}
		log.Logger.V(0).Info("OpAmp server exited")
		return err
	})

	// start kube manager
	g.Go(func() error {
		err := o.mgr.Start(groupCtx)
		if err != nil {
			log.Logger.Error(err, "error starting kube manager")
		} else {
			log.Logger.V(0).Info("Kube manager exited")
		}
		// the manager is stopped, it is now safe to close the config updates channel
		if o.configUpdates != nil {
			close(o.configUpdates)
		}
		return err
	})

	err := g.Wait()
	if err != nil {
		log.Logger.Error(err, "Odiglet exited with error")
	}
}

func OdigletInitPhase(clientset *kubernetes.Clientset) {
	odigletInitPhaseStart := time.Now()
	defer func() {
		log.Logger.V(0).Info("Odiglet init phase finished", "duration", time.Since(odigletInitPhaseStart))
	}()

	// Initialize logging
	if err := log.Init(); err != nil {
		panic(err)
	}

	// Step 1: Copy instrumentation agents to host
	if err := copyAgentsToHost(); err != nil {
		log.Logger.Error(err, "Failed to copy agents directory to host")
		os.Exit(-1)
	}

	// Step 2: Prepare node for Odigos installation (labels, taints)
	if err := prepareNode(clientset); err != nil {
		log.Logger.Error(err, "Failed to prepare node for Odigos installation")
		os.Exit(-1)
	}

	// Step 3: Apply SELinux settings (must be last - uses chroot)
	if err := applySecuritySettings(); err != nil {
		log.Logger.Error(err, "Failed to apply SELinux settings on RHEL host")
		os.Exit(-1)
	}

	os.Exit(0)
}

// copyAgentsToHost copies instrumentation agents from the odiglet image to the host filesystem.
func copyAgentsToHost() error {
	return fs.CopyAgentsDirectoryToHost()
}

// prepareNode prepares the Kubernetes node for Odigos installation by updating labels and taints.
func prepareNode(clientset *kubernetes.Clientset) error {
	nodeName, ok := os.LookupEnv(k8sconsts.NodeNameEnvVar)
	if !ok {
		return fmt.Errorf("env var %s is not set", k8sconsts.NodeNameEnvVar)
	}

	err := k8snode.PrepareNodeForOdigosInstallation(clientset, nodeName)
	if err != nil {
		return err
	}

	log.Logger.Info("Successfully prepared node for Odigos installation")
	return nil
}

// applySecuritySettings applies security settings like SELinux contexts.
// This must be called last as it uses chroot to access host commands.
func applySecuritySettings() error {
	return fs.ApplyOpenShiftSELinuxSettings()
}

// OdigletDiscoveryPhase runs the node discovery phase to collect and persist node details.
// This is run as a separate container/mode to collect node characteristics and capabilities.
// It checks all registered features (OSS + enterprise extensions) and creates a NodeDetails CRD.
func OdigletDiscoveryPhase(config *rest.Config, clientset *kubernetes.Clientset) {
	discoveryPhaseStart := time.Now()
	defer func() {
		log.Logger.V(0).Info("Odiglet discovery phase finished", "duration", time.Since(discoveryPhaseStart))
	}()

	// Initialize logging
	if err := log.Init(); err != nil {
		panic(err)
	}

	log.Logger.V(0).Info("Starting odiglet discovery phase")

	// Get node name from environment
	nodeName, ok := os.LookupEnv(k8sconsts.NodeNameEnvVar)
	if !ok {
		log.Logger.Error(fmt.Errorf("env var %s is not set", k8sconsts.NodeNameEnvVar), "Failed to load env")
		os.Exit(-1)
	}

	// Get node object from Kubernetes
	ctx := context.Background()
	node, err := clientset.CoreV1().Nodes().Get(ctx, nodeName, metav1.GetOptions{})
	if err != nil {
		log.Logger.Error(err, "Failed to get node", "node", nodeName)
		os.Exit(-1)
	}

	// Collect and persist node details (OSS + enterprise features)
	if err := nodedetails.PrepareAndCollect(config, node); err != nil {
		log.Logger.Error(err, "Failed to check and persist node features")
		os.Exit(-1)
	}

	log.Logger.V(0).Info("Successfully collected and persisted node details", "node", nodeName)

	// Keep the process running to allow for future restarts via signal
	// We wait for SIGTERM/SIGINT to exit gracefully
	log.Logger.V(0).Info("Discovery phase completed, waiting for signal...")

	// Create a context that is canceled on termination signals
	// We handle SIGINT, SIGTERM, and SIGQUIT to ensure graceful shutdown
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	defer stop()

	<-ctx.Done()
	log.Logger.V(0).Info("Received termination signal, exiting discovery phase")
	os.Exit(0)
}
