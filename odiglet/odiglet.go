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
	"github.com/odigos-io/odigos/odiglet/pkg/kube"
	ebpfMetrics "github.com/odigos-io/odigos/odiglet/pkg/metrics"
	"github.com/odigos-io/odigos/odiglet/pkg/process"
	"github.com/odigos-io/odigos/opampserver/pkg/server"
	"golang.org/x/sync/errgroup"

	"github.com/odigos-io/odigos/common/fs"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"k8s.io/client-go/kubernetes"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
)

// Runnable is a named task started in Odiglet.Run's errgroup. Run should block until ctx is
// canceled. Return nil on normal shutdown. When PropagateErr is true, a non-nil error is
// returned to the errgroup and cancels the other runnables.
type Runnable struct {
	Name         string
	PropagateErr bool
	Run          func(ctx context.Context) error
}

type Odiglet struct {
	clientset     *kubernetes.Clientset
	mgr           controllerruntime.Manager
	ebpfManager   commonInstrumentation.Manager
	configUpdates chan<- commonInstrumentation.ConfigUpdate[ebpf.K8sConfigGroup]
	// instrumentationRequests is kept as a bidirectional channel so we can hand out the
	// write-end to external producers via InstrumentationRequests() while also reading from it
	// in the embedded instrumentation manager.
	instrumentationRequests chan commonInstrumentation.Request[ebpf.K8sProcessGroup, ebpf.K8sConfigGroup, *ebpf.K8sProcessDetails]
	criClient               *criwrapper.CriClient
	runnables               []Runnable
}

// AddRunnable registers a task to run inside Odiglet.Run's errgroup. Call before Run; safe to
// call multiple times to register several tasks.
func (o *Odiglet) AddRunnable(r Runnable) {
	o.runnables = append(o.runnables, r)
}

// goRunnable runs r in the odiglet errgroup with standard logging derived from r.Name.
func (o *Odiglet) goRunnable(g *errgroup.Group, ctx context.Context, logger *commonlogger.OdigosLogger, r Runnable) {
	g.Go(func() error {
		err := r.Run(ctx)
		if err != nil {
			logger.Error("failed to run "+r.Name, "err", err)
		} else {
			logger.Info(r.Name + " exited")
		}
		if r.PropagateErr {
			return err
		}
		return nil
	})
}

// InstrumentationRequests returns the write-end of the instrumentation manager's request
// channel. External producers (e.g. the enterprise odiglet's Go offsets file watcher) can use
// this to send instrumentation, un-instrumentation, or retry-failed requests alongside the OSS
// odiglet's own kube reconcilers. See instrumentation.Request for the encoding of each request
// kind.
//
// Callers must use a non-blocking send (select with a default branch) and must NOT close the
// returned channel.
func (o *Odiglet) InstrumentationRequests() ebpf.K8sInstrumentationRequests {
	return o.instrumentationRequests
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

	process.DiscoverCgroupLayout()

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

	ebpfLogger := commonlogger.LoggerCompat().With("subsystem", "ebpfmanager")
	metricsLogger := commonlogger.LoggerCompat().With("subsystem", "ebpfmetrics")
	collector := ebpfMetrics.NewEBPFMetricsCollector(env.Current.NodeName, metricsLogger)
	if err := collector.RegisterMetrics(); err != nil {
		metricsLogger.Error("failed to register metrics", "err", err)
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
	ebpfManager, err := ebpf.NewManager(mgr.GetClient(), ebpfLogger, instrumentationMgrOpts, configUpdates, instrumentationRequests, appendEnvVarNames)
	if err != nil {
		return nil, fmt.Errorf("failed to create ebpf manager %w", err)
	}
	criWrapper := criwrapper.CriClient{Logger: commonlogger.ToLogr()}

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

func (o *Odiglet) builtInRunnables(ebpfDone chan struct{}, logger *commonlogger.OdigosLogger) []Runnable {
	odigosNs := env.GetCurrentNamespace()
	return []Runnable{
		{
			Name: "pprof server",
			// if we fail to start the pprof server, don't return an error as it is not critical
			PropagateErr: false,
			Run: func(ctx context.Context) error {
				return common.StartPprofServer(ctx, commonlogger.ToLogr(), int(k8sconsts.DefaultPprofEndpointPort))
			},
		},
		{
			Name:         "eBPF manager",
			PropagateErr: true,
			Run: func(ctx context.Context) error {
				defer close(ebpfDone)
				return o.ebpfManager.Run(ctx)
			},
		},
		{
			Name:         "OpAmp server",
			PropagateErr: true,
			Run: func(ctx context.Context) error {
				return server.StartOpAmpServer(ctx, o.mgr, o.clientset, env.Current.NodeName, odigosNs)
			},
		},
		{
			Name:         "kube manager",
			PropagateErr: true,
			Run: func(ctx context.Context) error {
				// Create a context that will be cancelled when eBPF manager exits during shutdown
				kubeManagerCtx, kubeManagerCancel := context.WithCancel(context.Background())
				defer kubeManagerCancel()

				go func() {
					select {
					case <-ctx.Done():
						logger.Info("Shutdown initiated, waiting for eBPF manager to exit before stopping kube manager")
						<-ebpfDone
						logger.Info("eBPF manager exited, now stopping kube manager")
						kubeManagerCancel()
					case <-kubeManagerCtx.Done():
						return
					}
				}()

				err := o.mgr.Start(kubeManagerCtx)
				if o.configUpdates != nil {
					close(o.configUpdates)
				}
				// We don't close instrumentationRequests here: the manager's runEventLoop returns on
				// ctx.Done() so it doesn't need a channel close to terminate, and external producers
				// can obtain the write-end via InstrumentationRequests(). Closing would race with
				// those producers (including the OSS kube reconcilers themselves if they're mid-send).
				return err
			},
		},
	}
}

// Run starts the Odiglet components and blocks until the context is cancelled, or a critical error occurs.
func (o *Odiglet) Run(ctx context.Context) {
	logger := commonlogger.LoggerCompat().With("subsystem", "eventloop")
	g, groupCtx := errgroup.WithContext(ctx)

	if err := o.criClient.Connect(ctx); err != nil {
		logger.Error("Failed to connect to CRI runtime", "err", err)
	}

	defer o.criClient.Close()

	// Channel to signal when eBPF manager has exited
	ebpfDone := make(chan struct{})
	runnables := append(o.builtInRunnables(ebpfDone, logger), o.runnables...)
	for _, runnable := range runnables {
		o.goRunnable(g, groupCtx, logger, runnable)
	}

	err := g.Wait()
	if err != nil {
		logger.Error("Odiglet exited with error", "err", err)
	}
}

func OdigletInitPhase(clientset *kubernetes.Clientset) {
	odigletInitPhaseStart := time.Now()
	// Logger already initialized in main() before calling OdigletInitPhase.
	logger := commonlogger.LoggerCompat().With("subsystem", "init")

	err := fs.CopyAgentsDirectoryToHost(k8sconsts.OdigletContainerAgentDirectory, k8sconsts.OdigosAgentsDirectory, nil)
	if err != nil {
		logger.Error("Failed to copy agents directory to host", "err", err)
		os.Exit(-1)
	}

	// Deterministically stage the native memprof agent libs (traversable 0755 dir +
	// canonical libjemalloc.so name) so non-root C/C++/Rust apps can LD_PRELOAD them.
	// Non-fatal: memory profiling degrades, the rest of odiglet must still come up.
	if err := fs.EnsureMemprofAgentLibs(k8sconsts.OdigletContainerAgentDirectory, k8sconsts.OdigosAgentsDirectory); err != nil {
		logger.Error("Failed to stage memprof agent libs", "err", err)
	}

	// Make every agent dir traversable so security-hardened (runAsNonRoot) workloads
	// can read the agents we mount in — a 0644 agent dir crashes a non-root JVM
	// ("JAR manifest missing") or breaks LD_PRELOAD even with the file present.
	if err := fs.EnsureAgentDirsTraversable(k8sconsts.OdigosAgentsDirectory); err != nil {
		logger.Error("Failed to make agent dirs traversable", "err", err)
	}

	nn, ok := os.LookupEnv(k8sconsts.NodeNameEnvVar)
	if !ok {
		logger.Error("Failed to load env", "err", fmt.Errorf("env var %s is not set", k8sconsts.NodeNameEnvVar))
		os.Exit(-1)
	}

	if k8snode.IsGKEManagedNode(nn) {
		logger.Info("Skipping node label setup for GKE Autopilot")
	} else if err := k8snode.PrepareNodeForOdigosInstallation(clientset, nn); err != nil {
		logger.Error("Failed to prepare node for Odigos installation", "err", err)
		os.Exit(-1)
	} else {
		logger.Info("Successfully prepared node for Odigos installation")
	}

	// SELinux settings should be applied last. This function chroot's to use the host's PATH for
	// executing selinux commands to make agents readable by pods.
	if err := fs.ApplyOpenShiftSELinuxSettings(k8sconsts.OdigosAgentsDirectory); err != nil {
		logger.Error("Failed to apply SELinux settings on RHEL host", "err", err)
		os.Exit(-1)
	}

	logger.Info("Odiglet init phase finished", "duration", time.Since(odigletInitPhaseStart))
	os.Exit(0)
}
