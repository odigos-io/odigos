/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled"
	"github.com/odigos-io/odigos/instrumentor/controllers/instrumentationconfig"
	"github.com/odigos-io/odigos/instrumentor/controllers/startlangdetection"
	"github.com/odigos-io/odigos/instrumentor/controllers/workloadmigrations"
	"github.com/odigos-io/odigos/instrumentor/sdks"

	corev1 "k8s.io/api/core/v1"

	runtimemigration "github.com/odigos-io/odigos/instrumentor/runtimemigration"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"github.com/odigos-io/odigos/instrumentor/controllers/deleteinstrumentationconfig"
	"github.com/odigos-io/odigos/instrumentor/report"
	"github.com/odigos-io/odigos/instrumentor"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports

	"net/http"
	_ "net/http/pprof"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(odigosv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var telemetryDisabled bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&telemetryDisabled, "telemetry-disabled", false, "Disable telemetry")

	opts := ctrlzap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	zapLogger := ctrlzap.NewRaw(ctrlzap.UseFlagOptions(&opts))
	zapLogger = bridge.AttachToZapLogger(zapLogger)
	logger := zapr.NewLogger(zapLogger)
	ctrl.SetLogger(logger)

	odigosNs := env.GetCurrentNamespace()
	nsSelector := client.InNamespace(odigosNs).AsSelector()
	odigosEffectiveConfigNameSelector := fields.OneTermEqualSelector("metadata.name", consts.OdigosEffectiveConfigName)
	odigosEffectiveConfigSelector := fields.AndSelectors(nsSelector, odigosEffectiveConfigNameSelector)
	instrumentedPodReq, _ := labels.NewRequirement(k8sconsts.OdigosAgentsMetaHashLabel, selection.Exists, []string{})
	instrumentedPodSelector := labels.NewSelector().Add(*instrumentedPodReq)

	podsTransformFunc := func(obj interface{}) (interface{}, error) {
		pod, ok := obj.(*corev1.Pod)
		if !ok {
			return nil, fmt.Errorf("expected a Pod, got %T", obj)
		}

		stripedStatus := corev1.PodStatus{
			Phase:             pod.Status.Phase,
			ContainerStatuses: pod.Status.ContainerStatuses, // TODO: we don't need all data here
			Message:           pod.Status.Message,
			Reason:            pod.Status.Reason,
			StartTime:         pod.Status.StartTime,
		}
		strippedPod := corev1.Pod{
			ObjectMeta: pod.ObjectMeta,
			Status:     stripedStatus,
		}
		strippedPod.SetManagedFields(nil) // don't store managed fields in the cache
		return &strippedPod, nil
	}

	mgrOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "201bdfa0.odigos.io",
		/*
			Leader Election Parameters:

			LeaseDuration (5s):
			- Maximum time a pod can remain the leader after its last successful renewal.
			- If the leader pod dies, failover can take up to the LeaseDuration from the last renewal.
			  The actual failover time depends on how recently the leader renewed the lease.
			- Controls when the lease is fully expired and failover can occur.

			RenewDeadline (4s):
			- The maximum time the leader pod has to successfully renew its lease before it is
			  considered unhealthy. Relevant only while the leader is alive and renewing.
			- Controls how long the current leader will keep retrying to refresh the lease.

			RetryPeriod (1s):
			- How often non-leader pods check and attempt to acquire leadership when the lease is available.
			- Lower value means faster failover but adds more load on the Kubernetes API server.

			Relationship:
			- RetryPeriod < RenewDeadline < LeaseDuration
			- This ensures proper failover timing and system stability.
		*/
		LeaseDuration: durationPointer(5 * time.Second),
		RenewDeadline: durationPointer(4 * time.Second),
		RetryPeriod:   durationPointer(1 * time.Second),
		Cache: cache.Options{
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				&corev1.Pod{}: {
					Label:     instrumentedPodSelector,
					Transform: podsTransformFunc,
				},
				&corev1.ConfigMap{}: {
					Field: odigosEffectiveConfigSelector,
				},
				&odigosv1.CollectorsGroup{}: {
					Field: nsSelector,
				},
				&odigosv1.Destination{}: {
					Field: nsSelector,
				},
				&odigosv1.InstrumentationRule{}: {
					Field: nsSelector,
				},
				&odigosv1.InstrumentationConfig{}: {
					// all instrumentation configs are managed by this controller
					// and should be pulled into the cache
				},
			},
		},
	}

	// Check if the environment variable `LOCAL_WEBHOOK_CERT_DIR` is set.
	// If defined, add WebhookServer options with the specified certificate directory.
	// This is used primarily for local development environments to provide a custom path for serving TLS certificates.
	localCertDir := os.Getenv("LOCAL_MUTATING_WEBHOOK_CERT_DIR")
	if localCertDir != "" {
		mgrOptions.WebhookServer = webhook.NewServer(webhook.Options{
			CertDir: localCertDir,
		})
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)

	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	ctx := signals.SetupSignalHandler()

	// This temporary migration step ensures the runtimeDetails migration in the instrumentationConfig is performed.
	// This code can be removed once the migration is confirmed to be successful.
	mgr.Add(&runtimemigration.MigrationRunnable{KubeClient: mgr.GetClient(), Logger: setupLog})

	err = sdks.SetDefaultSDKs(ctx)

	if err != nil {
		setupLog.Error(err, "Failed to set default SDKs")
		os.Exit(-1)
	}

	err = agentenabled.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	err = deleteinstrumentationconfig.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	err = startlangdetection.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	err = instrumentationconfig.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller for instrumentation rules")
		os.Exit(1)
	}

	err = workloadmigrations.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller for instrumentation label migration")
		os.Exit(1)
	}

	err = builder.
		WebhookManagedBy(mgr).
		For(&odigosv1.Source{}).
		WithDefaulter(&instrumentor.SourcesDefaulter{
			Client: mgr.GetClient(),
		}).
		WithValidator(&instrumentor.SourcesValidator{
			Client: mgr.GetClient(),
		}).
		Complete()
	if err != nil {
		setupLog.Error(err, "unable to create Sources webhooks")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	err = addHealthAndReadyChecks(mgr)
	if err != nil {
		setupLog.Error(err, "unable to set up health and ready checks")
		os.Exit(1)
	}

	go common.StartPprofServer(ctx, setupLog)

	if !telemetryDisabled {
		go report.Start(mgr.GetClient())
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func addHealthAndReadyChecks(mgr ctrl.Manager) error {
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("unable to set up health check: %w", err)
	}

	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error{
		return mgr.GetWebhookServer().StartedChecker()(req)
	}); err != nil {
		return fmt.Errorf("unable to set up ready check: %w", err)
	}
	return nil
}

func durationPointer(d time.Duration) *time.Duration {
	return &d
}
