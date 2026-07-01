package server

import (
	"context"
	"fmt"
	"sync"

	"github.com/go-logr/logr"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/odigos-io/odigos/common/consts"
	commonlogger "github.com/odigos-io/odigos/common/logger"
	"github.com/odigos-io/odigos/config"
	"github.com/odigos-io/odigos/actions"
	"github.com/odigos-io/odigos/destinations"
	"github.com/odigos-io/odigos/instrumentationrules"
	"github.com/odigos-io/odigos/frontend/kube"
	"github.com/odigos-io/odigos/frontend/kube/watchers"
	"github.com/odigos-io/odigos/frontend/services"
	collectormetrics "github.com/odigos-io/odigos/frontend/services/collector_metrics"
	"github.com/odigos-io/odigos/frontend/services/metrics"
	"github.com/odigos-io/odigos/frontend/services/otlp"
	"github.com/odigos-io/odigos/frontend/services/profiles"
	"github.com/odigos-io/odigos/frontend/services/tracecorrelations"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
)

// Deps is the dependency bundle the frontend HTTP layer (and any extra mounts
// like the enterprise MCP server) needs. Constructed once at startup by
// Bootstrap and threaded through the rest of the lifecycle.
type Deps struct {
	Flags         Flags
	Logger        logr.Logger
	K8sCacheClient client.Client
	K8sCache       cache.Cache

	OdigosMetrics               *collectormetrics.OdigosMetricsConsumer
	PromAPI                     v1.API
	CorrelationsPromAPI         v1.API
	CorrelationsMetricsStoreURL string
	ProfileStore     *profiles.ProfileStore
	ProfilingGate    *profiles.IngestGate
	ProfilesConsumer *profiles.OdigosProfilesConsumer
	OtlpReceiver     *otlp.Receiver
}

// Bootstrap performs the synchronous startup work: load embedded destination
// + config catalogs, create the kube client + cache, set up the OTLP receiver
// and metrics/profile consumers, and resolve the initial profiling state
// from effective-config. Does NOT start any goroutines — call StartBackground
// for that. Splitting these means out-of-tree mains can interleave their own
// boot steps (e.g. AssertEnterpriseOdigosToken) between the two phases.
func Bootstrap(ctx context.Context, flags Flags, logger logr.Logger) (*Deps, error) {
	log := commonlogger.LoggerCompat().With("subsystem", "bootstrap")

	if err := destinations.Load(); err != nil {
		return nil, fmt.Errorf("loading destinations data: %w", err)
	}
	if err := actions.Load(); err != nil {
		return nil, fmt.Errorf("loading actions data: %w", err)
	}
	if err := instrumentationrules.Load(); err != nil {
		return nil, fmt.Errorf("loading instrumentation rules data: %w", err)
	}
	if err := config.Load(); err != nil {
		return nil, fmt.Errorf("loading config data: %w", err)
	}

	if err := initKubernetesClient(flags); err != nil {
		return nil, err
	}

	// Source cache: controller-runtime cache for fast in-process reads across namespaces.
	k8sCacheClient, k8sCache, err := kube.SetupK8sCache(ctx, flags.KubeConfig, flags.KubeContext, flags.Namespace)
	if err != nil {
		return nil, fmt.Errorf("setting up k8s objects cache: %w", err)
	}

	odigosMetrics := collectormetrics.NewOdigosMetrics()

	// Profiling state derives from effective-config. Failure is non-fatal —
	// the IngestGate stays off until the watcher sees a usable config.
	profCfg, profCfgErr := services.ResolveProfilingFromEffectiveConfig(ctx, k8sCacheClient)
	profilingIngest := false
	if profCfgErr == nil {
		profilingIngest = profCfg.ReceiverOn
	} else {
		log.Error("profiling: could not load initial effective config; ingest off until effective-config is readable", "err", profCfgErr)
	}
	profilingGate := profiles.NewProfilesIngestGate(profilingIngest)
	profileStore := profiles.NewProfileStore(
		profCfg.StoreLimits.MaxSlots,
		profCfg.StoreLimits.SlotTTLSeconds,
		profCfg.StoreLimits.SlotMaxBytes,
		profCfg.CleanupInterval,
	)
	profileStore.RunCleanup(ctx)

	profilesConsumer, err := profiles.NewOdigosProfilesConsumer(profileStore, profilingGate)
	if err != nil {
		log.Warn("profiles consumer init failed", "err", err)
		profilesConsumer = nil
	}

	otlpReceiver, err := otlp.NewReceiver(consts.OTLPPort)
	if err != nil {
		log.Warn("otlpReceiver config failed", "err", err)
		otlpReceiver = nil
	}

	// VictoriaMetrics-backed Prometheus API for per-pod collector metrics
	// surfaced through the GraphQL/Odiglet UI panes.
	var promAPI v1.API
	metricsURL := fmt.Sprintf("http://%s.%s.svc:8428", metrics.VictoriaMetricsServiceName, flags.Namespace)
	if api, err := metrics.NewAPIFromURL(metricsURL); err != nil {
		log.Warn("failed to initialize VictoriaMetrics API", "url", metricsURL, "err", err)
	} else {
		promAPI = api
	}

	var correlationsPromAPI v1.API
	correlationsURL := tracecorrelations.MetricsStoreURL(flags.Namespace)
	if api, err := metrics.NewAPIFromURL(correlationsURL); err != nil {
		log.Warn("failed to initialize trace correlations VictoriaMetrics API", "url", correlationsURL, "err", err)
	} else {
		correlationsPromAPI = api
	}

	return &Deps{
		Flags:                       flags,
		Logger:                      logger,
		K8sCacheClient:              k8sCacheClient,
		K8sCache:                    k8sCache,
		OdigosMetrics:               odigosMetrics,
		PromAPI:                     promAPI,
		CorrelationsPromAPI:         correlationsPromAPI,
		CorrelationsMetricsStoreURL: correlationsURL,
		ProfileStore:                profileStore,
		ProfilingGate:               profilingGate,
		ProfilesConsumer:            profilesConsumer,
		OtlpReceiver:                otlpReceiver,
	}, nil
}

// StartBackground launches the long-running goroutines: the metrics-consumer
// delete watcher, the OTLP receiver lifecycle, and the source/destination/
// profiling-config watchers. Returns a WaitGroup the caller must Wait on at
// shutdown (after cancelling the context).
func StartBackground(ctx context.Context, deps *Deps) (*sync.WaitGroup, error) {
	var wg sync.WaitGroup

	// Metrics consumer's K8s delete watcher + notification loop.
	wg.Add(1)
	go func() {
		defer wg.Done()
		deps.OdigosMetrics.RunDeleteWatcherAndNotifications(ctx, deps.Flags.Namespace)
	}()

	// OTLP receiver: warn-and-continue if it can't start (UI still works,
	// just without live ingest of metrics/profiles).
	if deps.OtlpReceiver != nil {
		pipelines := []otlp.OTLPPipeline{
			otlp.NewMetricsPipeline(deps.OtlpReceiver, deps.OdigosMetrics),
		}
		if deps.ProfilesConsumer != nil {
			pipelines = append(pipelines, otlp.NewProfilesPipeline(deps.OtlpReceiver, deps.ProfilesConsumer.GetConsumer()))
		}
		if err := deps.OtlpReceiver.Start(ctx, pipelines...); err != nil {
			commonlogger.LoggerCompat().With("subsystem", "bootstrap").Warn("OTLP setup failed", "err", err)
		} else {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if err := deps.OtlpReceiver.WaitAndShutdown(ctx, pipelines...); err != nil {
					commonlogger.LoggerCompat().With("subsystem", "bootstrap").Error("OTLP receiver shutdown failed", "err", err)
				}
			}()
		}
	}

	// In-cluster watchers (Source/Destination/Profiling).
	var ingestGate *profiles.IngestGate
	var store *profiles.ProfileStore
	if deps.ProfilesConsumer != nil {
		ingestGate = deps.ProfilingGate
		store = deps.ProfileStore
	}
	if err := startWatchers(ctx, deps.K8sCache, deps.OdigosMetrics, ingestGate, store); err != nil {
		return &wg, err
	}

	return &wg, nil
}

// initKubernetesClient creates and installs the default kube client + workload
// kind availability map. Kept package-private — out-of-tree mains call
// Bootstrap (which calls this).
func initKubernetesClient(flags Flags) error {
	c, err := kube.CreateClient(flags.KubeConfig, flags.KubeContext)
	if err != nil {
		return fmt.Errorf("creating Kubernetes client: %w", err)
	}
	kube.SetDefaultClient(c)
	kube.InitWorkloadKindsAvailability()
	return nil
}

func startWatchers(
	ctx context.Context,
	k8sCache cache.Cache,
	odigosMetrics *collectormetrics.OdigosMetricsConsumer,
	profilingGate *profiles.IngestGate,
	profileStore *profiles.ProfileStore,
) error {
	if err := watchers.StartInstrumentationConfigWatcher(ctx, k8sCache, odigosMetrics); err != nil {
		return fmt.Errorf("starting InstrumentationConfig watcher: %w", err)
	}
	if err := watchers.StartDestinationWatcher(ctx, k8sCache, odigosMetrics); err != nil {
		return fmt.Errorf("starting Destination watcher: %w", err)
	}
	if profilingGate != nil && profileStore != nil {
		if err := watchers.StartProfilingConfigWatcher(ctx, k8sCache, env.GetCurrentNamespace(), profilingGate, profileStore); err != nil {
			return fmt.Errorf("starting profiling effective-config watcher: %w", err)
		}
	}
	return nil
}
