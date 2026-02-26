package odiglet

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	commonlogger "github.com/odigos-io/odigos/common/logger"
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
	ebpfMetrics "github.com/odigos-io/odigos/odiglet/pkg/metrics"
	"github.com/odigos-io/odigos/opampserver/pkg/server"
	"golang.org/x/sync/errgroup"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/client-go/kubernetes"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

type Odiglet struct {
	clientset               *kubernetes.Clientset
	mgr                     controllerruntime.Manager
	ebpfManager             commonInstrumentation.Manager
	configUpdates           chan<- commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	instrumentationRequests chan<- commonInstrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails]
	criClient               *criwrapper.CriClient
}

// channel sizes for sending events to the instrumentation manager's event loop.
// during bursts, or start-ups we want to be able to queue events in the channels without blocking the reconciler.
const (
	configUpdatesBufferSize           = 100
	instrumentationRequestsBufferSize = 200
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

	logger := commonlogger.Logger()
	collector := ebpfMetrics.NewEBPFMetricsCollector(env.Current.NodeName)
	if err := collector.RegisterMetrics(); err != nil {
		logger.Error("failed to register metrics", "err", err)
	}

	appendEnvVarNames := distro.GetAppendEnvVarNames(instrumentationMgrOpts.DistributionGetter.GetAllDistros())

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up health check: %w", err)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up ready check: %w", err)
	}

	configUpdates := make(chan commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup], configUpdatesBufferSize)
	instrumentationRequests := make(chan commonInstrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails], instrumentationRequestsBufferSize)
	ebpfManager, err := ebpf.NewManager(mgr.GetClient(), instrumentationMgrOpts, configUpdates, instrumentationRequests, appendEnvVarNames)
	if err != nil {
		return nil, fmt.Errorf("failed to create ebpf manager %w", err)
	}
	criWrapper := criwrapper.CriClient{}

	kubeManagerOptions := kube.KubeManagerOptions{
		Mgr:                     mgr,
		ConfigUpdates:           configUpdates,
		InstrumentationRequests: instrumentationRequests,
		CriClient:               &criWrapper,
		AppendEnvVarNames:       appendEnvVarNames,
	}

	err = kube.SetupWithManager(kubeManagerOptions, instrumentationMgrOpts.DistributionGetter)
	if err != nil {
		return nil, fmt.Errorf("failed to setup controller-runtime manager %w", err)
	}

	return &Odiglet{
		clientset:               clientset,
		mgr:                     mgr,
		ebpfManager:             ebpfManager,
		configUpdates:           configUpdates,
		instrumentationRequests: instrumentationRequests,
		criClient:               &criWrapper,
	}, nil
}

// Run starts the Odiglet components and blocks until the context is cancelled, or a critical error occurs.
func (o *Odiglet) Run(ctx context.Context) {
	logger := commonlogger.Logger()
	g, groupCtx := errgroup.WithContext(ctx)

	if err := o.criClient.Connect(ctx); err != nil {
		logger.Error("Failed to connect to CRI runtime", "err", err)
	}

	defer o.criClient.Close()

	// Channel to signal when eBPF manager has exited
	ebpfDone := make(chan struct{})

	// Start debug server
	g.Go(func() error {
		err := common.StartDebugServer(groupCtx, logger, int(k8sconsts.DefaultDebugPort))
		if err != nil {
			logger.Error("Failed to start debug server", "err", err)
		} else {
			logger.Info("Debug server exited")
		}
		// if we fail to start the debug server, don't return an error as it is not critical
		// and we can run the rest of the components
		return nil
	})

	g.Go(func() error {
		defer close(ebpfDone)
		err := o.ebpfManager.Run(groupCtx)
		if err != nil {
			logger.Error("Failed to run ebpf manager", "err", err)
		}
		logger.Info("eBPF manager exited")
		return err
	})

	// start OpAmp server
	odigosNs := env.GetCurrentNamespace()
	g.Go(func() error {
		err := server.StartOpAmpServer(groupCtx, o.mgr, o.clientset, env.Current.NodeName, odigosNs)
		if err != nil {
			logger.Error("Failed to start opamp server", "err", err)
		}
		logger.Info("OpAmp server exited")
		return err
	})

	// start kube manager
	g.Go(func() error {
		// Create a context that will be cancelled when eBPF manager exits during shutdown
		kubeManagerCtx, kubeManagerCancel := context.WithCancel(context.Background())
		defer kubeManagerCancel()

		go func() {
			select {
			case <-groupCtx.Done():
				logger.Info("Shutdown initiated, waiting for eBPF manager to exit before stopping kube manager")
				<-ebpfDone
				logger.Info("eBPF manager exited, now stopping kube manager")
				kubeManagerCancel()
			case <-kubeManagerCtx.Done():
				// Kube context already cancelled
				return
			}
		}()

		err := o.mgr.Start(kubeManagerCtx)
		if err != nil {
			logger.Error("error starting kube manager", "err", err)
		} else {
			logger.Info("Kube manager exited")
		}
		// the manager is stopped, it is now safe to close the config updates channel
		if o.configUpdates != nil {
			close(o.configUpdates)
		}
		if o.instrumentationRequests != nil {
			close(o.instrumentationRequests)
		}
		return err
	})

	err := g.Wait()
	if err != nil {
		logger.Error("Odiglet exited with error", "err", err)
	}
}

func OdigletInitPhase(clientset *kubernetes.Clientset) {
	odigletInitPhaseStart := time.Now()
	// Logger already initialized in main() before calling OdigletInitPhase.
	logger := commonlogger.Logger()

	err := fs.CopyAgentsDirectoryToHost()
	if err != nil {
		logger.Error("Failed to copy agents directory to host", "err", err)
		os.Exit(-1)
	}

	nn, ok := os.LookupEnv(k8sconsts.NodeNameEnvVar)
	if !ok {
		logger.Error("Failed to load env", "err", fmt.Errorf("env var %s is not set", k8sconsts.NodeNameEnvVar))
		os.Exit(-1)
	}

	if err := k8snode.PrepareNodeForOdigosInstallation(clientset, nn); err != nil {
		logger.Error("Failed to prepare node for Odigos installation", "err", err)
		os.Exit(-1)
	} else {
		logger.Info("Successfully prepared node for Odigos installation")
	}

	// SELinux settings should be applied last. This function chroot's to use the host's PATH for
	// executing selinux commands to make agents readable by pods.
	if err := fs.ApplyOpenShiftSELinuxSettings(); err != nil {
		logger.Error("Failed to apply SELinux settings on RHEL host", "err", err)
		os.Exit(-1)
	}

	logger.Info("Odiglet init phase finished", "duration", time.Since(odigletInitPhaseStart))
	os.Exit(0)
}
