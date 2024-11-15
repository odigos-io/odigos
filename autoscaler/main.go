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
	"context"
	"flag"
	"os"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	odigosver "github.com/odigos-io/odigos/k8sutils/pkg/version"

	corev1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/go-logr/logr"
	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/version"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	apiactions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	observabilitycontrolplanev1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"github.com/odigos-io/odigos/autoscaler/controllers"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	nameutils "github.com/odigos-io/odigos/autoscaler/utils"

	//+kubebuilder:scaffold:imports

	_ "net/http/pprof"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(observabilitycontrolplanev1.AddToScheme(scheme))
	utilruntime.Must(apiactions.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var imagePullSecretsString string
	var imagePullSecrets []string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&imagePullSecretsString, "image-pull-secrets", "",
		"The image pull secrets to use for the collectors created by autoscaler")
	flag.StringVar(&nameutils.ImagePrefix, "image-prefix", "", "The image prefix to use for the collectors created by autoscaler")

	odigosVersion := os.Getenv("ODIGOS_VERSION")
	if odigosVersion == "" {
		flag.StringVar(&odigosVersion, "version", "", "for development purposes only")
	}
	// Get k8s version
	k8sVersion, err := odigosver.GetKubernetesVersion()
	if err != nil {
		setupLog.Error(err, "unable to get Kubernetes version, continuing with default oldest supported version")
	}

	opts := ctrlzap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	if imagePullSecretsString != "" {
		imagePullSecrets = strings.Split(imagePullSecretsString, ",")
	}

	zapLogger := ctrlzap.NewRaw(ctrlzap.UseFlagOptions(&opts))
	zapLogger = bridge.AttachToZapLogger(zapLogger)
	logger := zapr.NewLogger(zapLogger)
	ctrl.SetLogger(logger)

	if odigosVersion == "" {
		setupLog.Error(nil, "ODIGOS_VERSION environment variable is not set and version flag is not provided")
		os.Exit(1)
	}

	go common.StartPprofServer(setupLog)

	setupLog.Info("Starting odigos autoscaler", "version", odigosVersion)
	odigosNs := env.GetCurrentNamespace()
	nsSelector := client.InNamespace(odigosNs).AsSelector()
	clusterCollectorLabelSelector := labels.Set(gateway.ClusterCollectorGateway).AsSelector()

	cfg := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		Cache: cache.Options{
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				&appsv1.Deployment{}: {
					Label: clusterCollectorLabelSelector,
					Field: nsSelector,
				},
				&corev1.Service{}: {
					Label: clusterCollectorLabelSelector,
					Field: nsSelector,
				},
				&appsv1.DaemonSet{}: {
					Field: nsSelector,
				},
				&corev1.ConfigMap{}: {
					Field: nsSelector,
				},
				&corev1.Secret{}: {
					Field: nsSelector,
				},
			},
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "f681cfed.odigos.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}
	// Ver < 1.23
	var migrationClient client.Client
	// Determine if we need to use a non-caching client based on the Kubernetes version, version <1.23 has issues when using caching client

	// The labaling was for ver 1.0.91, migration is not releavant for old k8s versions which couln't run.
	// This is the reason we skip it for versions < 1.23 (Also, versions < 1.23 require a non-caching client and API chane)
	if k8sVersion != nil && k8sVersion.GreaterThan(version.MustParse("v1.23")) {
		// Use the cached client for versions >= 1.23
		err = MigrateCollectorsWorkloadToNewLabels(context.Background(), mgr.GetClient(), odigosNs)
		if err != nil {
			setupLog.Error(err, "unable to migrate collectors workload to new labels")
			os.Exit(1)
		}
	}

	// The name processor is used to transform device ids injected with the virtual device,
	// to service names and k8s attributes.
	// it is not needed for eBPF instrumentation or OpAMP implementations.
	// at the time of writing (2024-10-22) only dotnet and java native agent are using the name processor.
	_, disableNameProcessor := os.LookupEnv("DISABLE_NAME_PROCESSOR")

	config := &controllerconfig.ControllerConfig{
		MetricsServerEnabled: isMetricsServerInstalled(mgr, setupLog),
	}

	if err = (&controllers.DestinationReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
		Config:           config,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Destination")
		os.Exit(1)
	}

	if err = (&controllers.ProcessorReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		ImagePullSecrets:     imagePullSecrets,
		OdigosVersion:        odigosVersion,
		K8sVersion:           k8sVersion,
		DisableNameProcessor: disableNameProcessor,
		Config:               config,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Processor")
		os.Exit(1)
	}
	if err = (&controllers.CollectorsGroupReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		ImagePullSecrets:     imagePullSecrets,
		OdigosVersion:        odigosVersion,
		K8sVersion:           k8sVersion,
		DisableNameProcessor: disableNameProcessor,
		Config:               config,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CollectorsGroup")
		os.Exit(1)
	}
	if err = (&controllers.InstrumentedApplicationReconciler{
		Client:               mgr.GetClient(),
		Scheme:               mgr.GetScheme(),
		ImagePullSecrets:     imagePullSecrets,
		OdigosVersion:        odigosVersion,
		K8sVersion:           k8sVersion,
		DisableNameProcessor: disableNameProcessor,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InstrumentedApplication")
		os.Exit(1)
	}
	if err = (&controllers.OdigosConfigReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
		Config:           config,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OdigosConfig")
		os.Exit(1)
	}
	if err = (&controllers.SecretReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
		Config:           config,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Secret")
		os.Exit(1)
	}
	if err = (&controllers.GatewayDeploymentReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Deployment")
		os.Exit(1)
	}
	if err = (&controllers.DataCollectionDaemonSetReconciler{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "DaemonSet")
		os.Exit(1)
	}

	if err = actions.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create odigos actions controllers")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func isMetricsServerInstalled(mgr ctrl.Manager, logger logr.Logger) bool {
	var metricsServerDeployment appsv1.Deployment
	// Use APIReader (uncached client) for direct access to the API server
	// uses because mgr not cache the metrics-server deployment
	err := mgr.GetAPIReader().Get(context.TODO(), types.NamespacedName{Name: "metrics-server", Namespace: "kube-system"}, &metricsServerDeployment)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Metrics-server deployment not found")
		} else {
			logger.Error(err, "Failed to get metrics-server deployment")
		}
		return false
	}

	logger.V(0).Info("Metrics server found")
	return true
}
