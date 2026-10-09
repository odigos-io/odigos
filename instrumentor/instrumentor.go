package instrumentor

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	googlecloudmetadata "cloud.google.com/go/compute/metadata"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/distros"
	"github.com/odigos-io/odigos/instrumentor/controllers"
	controllerconfig "github.com/odigos-io/odigos/instrumentor/controllers/controller_config"
	"github.com/odigos-io/odigos/instrumentor/controllers/metricshandler"
	"github.com/odigos-io/odigos/instrumentor/controllers/pipeline"
	"github.com/odigos-io/odigos/instrumentor/internal/clusterinfo"
	"github.com/odigos-io/odigos/instrumentor/report"
	"github.com/odigos-io/odigos/k8sutils/pkg/certs"
	"github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/recommendations"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

const defaultCollectorImage = "registry.odigos.io/odigos-collector"

type Instrumentor struct {
	mgr                controllerruntime.Manager
	certReady          chan struct{}
	dp                 *distros.Provider
	webhooksRegistered *atomic.Bool
	k8sVersion         *version.Version
}

// Options configures a new Instrumentor.
type Options struct {
	ManagerOptions  controllers.KubeManagerOptions
	DistrosProvider *distros.Provider
	// Runnables build runnables that are registered on the controller-runtime manager
	// before it starts (e.g. enterprise periodic jobs). Each factory receives the manager
	// so the runnable can use its cached client. Runnables participate in manager
	// lifecycle and can opt into leader election via manager.LeaderElectionRunnable.
	Runnables []func(mgr manager.Manager) manager.Runnable
}

func New(opts Options) (*Instrumentor, error) {
	err := feature.Setup()
	if err != nil {
		return nil, err
	}

	err = destinations.Load()
	if err != nil {
		return nil, fmt.Errorf("unable to load destinations data: %w", err)
	}

	if err := recommendations.Load(); err != nil {
		return nil, fmt.Errorf("unable to load recommendations catalog: %w", err)
	}

	mgr, err := controllers.CreateManager(opts.ManagerOptions)
	if err != nil {
		return nil, err
	}

	for _, newRunnable := range opts.Runnables {
		if err := mgr.Add(newRunnable(mgr)); err != nil {
			return nil, fmt.Errorf("unable to add runnable: %w", err)
		}
	}

	odigosNs := env.GetCurrentNamespace()

	// One-shot upgrade cleanup after autoscaler merged into instrumentor:
	// delete leftover autoscaler webhook cert Secrets and the custom-metrics
	// APIService if Odigos still owns it but Helm does not, so the next helm
	// upgrade can create it. Safe once those objects are gone.
	// Remove after 8 April 2027 (6 months after this landed).
	mgr.Add(&certs.SecretDeleteMigration{Client: mgr.GetClient(), Logger: opts.ManagerOptions.Logger, Secret: types.NamespacedName{
		Namespace: odigosNs,
		Name:      k8sconsts.DeprecatedAutoscalerWebhookSecretName,
	}})
	mgr.Add(&certs.SecretDeleteMigration{Client: mgr.GetClient(), Logger: opts.ManagerOptions.Logger, Secret: types.NamespacedName{
		Namespace: odigosNs,
		Name:      k8sconsts.AutoscalerWebhookSecretName,
	}})
	mgr.Add(&metricshandler.APIServiceDeleteMigration{
		Client: mgr.GetClient(),
		Logger: opts.ManagerOptions.Logger,
	})

	dynamicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("unable to create dynamic client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		return nil, fmt.Errorf("unable to create kubernetes client: %w", err)
	}
	err = clusterinfo.RecordClusterInfo(context.Background(), clientset, odigosNs)
	if err != nil {
		opts.ManagerOptions.Logger.Error(err, "unable to record cluster info, skipping")
	}

	// setup the certificate rotator
	rotatorSetupFinished := make(chan struct{})
	err = rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: env.GetCurrentNamespace(),
			Name:      k8sconsts.InstrumentorWebhookSecretName,
		},
		CertDir: filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs"),
		IsReady: rotatorSetupFinished,
		CAName:  k8sconsts.InstrumentorCAName,
		Webhooks: []rotator.WebhookInfo{
			{Name: k8sconsts.InstrumentorMutatingWebhookName, Type: rotator.Mutating},
			{Name: k8sconsts.InstrumentorSourceMutatingWebhookName, Type: rotator.Mutating},
			{Name: k8sconsts.InstrumentorSourceValidatingWebhookName, Type: rotator.Validating},
			{Name: k8sconsts.AutoscalerActionValidatingWebhookName, Type: rotator.Validating},
		},
		DNSName: "serving-cert",
		ExtraDNSNames: []string{
			fmt.Sprintf("%s.%s.svc", k8sconsts.InstrumentorServiceName, env.GetCurrentNamespace()),
			fmt.Sprintf("%s.%s.svc.cluster.local", k8sconsts.InstrumentorServiceName, env.GetCurrentNamespace()),
		},
		EnableReadinessCheck: true,

		// marking the controller as the owner of the webhooks config updated fields (caBundle)
		// this helps to avoid CI/CD systems overwriting the controller set fields.
		FieldOwner: k8sconsts.InstrumentorWebhookFieldOwner,

		// we could set RequireLeaderElection to true here but that will make the readiness probe fail for non-leader
		// instances (since the IsReady channel will not be closed in non-leader instances).

		// these are the defaults, but we set them explicitly for clarity
		CaCertDuration:         10 * 365 * 24 * time.Hour, // 10 years
		ServerCertDuration:     1 * 365 * 24 * time.Hour,  // 1 year
		RotationCheckFrequency: 12 * time.Hour,            // 12 hours
		LookaheadInterval:      90 * 24 * time.Hour,       // 90 days
	})
	if err != nil {
		return nil, fmt.Errorf("unable to add cert rotator: %w", err)
	}

	k8sVersion, err := utils.ClusterVersion()
	if err != nil {
		return nil, err
	}

	collectorImage := defaultCollectorImage
	if collectorImageEnv, ok := os.LookupEnv("ODIGOS_COLLECTOR_IMAGE"); ok {
		collectorImage = collectorImageEnv
	}
	onGKE := isRunningOnGKE(context.Background())
	if onGKE {
		opts.ManagerOptions.Logger.Info("Running on GKE")
	}
	pipeline.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion:     feature.K8sVersion(),
		CollectorImage: collectorImage,
		OnGKE:          onGKE,
	}

	// wire up the controllers and webhooks
	scheduleOdigletOnlyOnInstrumentedNodes, instrumentedPodsNodeLabelRetention := parseFirstInstrumentedPodAtNodeLabelRetention()
	configOpts := controllers.OdigosConfigurationOptions{
		Tier:          env.GetOdigosTierFromEnv(),
		OdigosVersion: os.Getenv(consts.OdigosVersionEnvVarName),
		DynamicClient: dynamicClient,
	}
	err = controllers.SetupWithManager(context.Background(), mgr, opts.DistrosProvider, k8sVersion, scheduleOdigletOnlyOnInstrumentedNodes, instrumentedPodsNodeLabelRetention, configOpts)
	if err != nil {
		return nil, err
	}

	// Add health and ready probes
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return nil, fmt.Errorf("unable to set up health check: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		return mgr.GetWebhookServer().StartedChecker()(req)
	}); err != nil {
		return nil, fmt.Errorf("unable to set up ready check: %w", err)
	}

	webhooksRegistered := &atomic.Bool{}
	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		if !webhooksRegistered.Load() {
			return errors.New("webhooks not registered yet")
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("unable to set up cert rotator check: %w", err)
	}

	return &Instrumentor{
		mgr:                mgr,
		certReady:          rotatorSetupFinished,
		dp:                 opts.DistrosProvider,
		webhooksRegistered: webhooksRegistered,
		k8sVersion:         k8sVersion,
	}, nil
}

func (i *Instrumentor) Run(ctx context.Context, odigosTelemetryDisabled bool) {
	logger := commonlogger.LoggerCompat().With("subsystem", "instrumentor")
	g, groupCtx := errgroup.WithContext(ctx)

	// Start pprof server
	g.Go(func() error {
		err := common.StartPprofServer(groupCtx, commonlogger.ToLogr(), int(k8sconsts.DefaultPprofEndpointPort))
		if err != nil {
			logger.Error("Failed to start pprof server", "err", err)
		} else {
			logger.Info("Pprof server exited")
		}
		// if we fail to start the pprof server, don't return an error as it is not critical
		// and we can run the rest of the components
		return nil
	})

	if !odigosTelemetryDisabled {
		// Start telemetry report
		g.Go(func() error {
			report.Start(groupCtx, i.mgr.GetClient())
			logger.Info("Telemetry reporting exited")
			return nil
		})
	}

	// start kube manager
	g.Go(func() error {
		err := i.mgr.Start(groupCtx)
		if err != nil {
			logger.Error("error starting kube manager", "err", err)
		} else {
			logger.Info("Kube manager exited")
		}
		return err
	})

	// register webhooks after the certificate is ready
	g.Go(func() error {
		select {
		case <-i.certReady:
		case <-groupCtx.Done():
			return nil
		}
		logger.Info("Cert rotator is ready")
		err := controllers.RegisterWebhooks(i.mgr, controllers.WebhookConfig{
			DistrosProvider: i.dp,
			K8sVersion:      i.k8sVersion,
		})
		if err != nil {
			return err
		}
		if err := metricshandler.RegisterCustomMetricsAPI(i.mgr); err != nil {
			logger.Error("failed to register custom metrics API", "err", err)
		} else {
			logger.Info("Custom Metrics API registered successfully")
		}
		i.webhooksRegistered.Store(true)
		logger.Info("Webhooks registered")
		return nil
	})

	err := g.Wait()
	if err != nil {
		logger.Error("Instrumentor exited with error", "err", err)
	}
}

func parseFirstInstrumentedPodAtNodeLabelRetention() (enabled bool, retention time.Duration) {
	raw := os.Getenv(k8sconsts.FirstInstrumentedPodAtNodeLabelRetentionEnvVar)
	if raw == "" {
		return false, 0
	}
	d, err := time.ParseDuration(raw)
	if err != nil {
		commonlogger.LoggerCompat().Error("invalid instrumented pods node label retention, using 5m",
			"env", k8sconsts.FirstInstrumentedPodAtNodeLabelRetentionEnvVar, "value", raw, "err", err)
		return true, 5 * time.Minute
	}
	if d < 0 {
		d = 0
	}
	return true, d
}

// based on https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/blob/19c4db6ea12211308fbd2cba12cc8665a5b7c890/detectors/gcp/gke.go#L34
func isRunningOnGKE(ctx context.Context) bool {
	c := googlecloudmetadata.NewClient(nil)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.InstanceAttributeValueWithContext(ctx, "cluster-location")
	return err == nil
}
