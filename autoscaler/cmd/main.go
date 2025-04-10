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
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"

	corev1 "k8s.io/api/core/v1"

	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"sigs.k8s.io/controller-runtime/pkg/cache"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	apiactions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"github.com/odigos-io/odigos/autoscaler/controllers"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"

	//+kubebuilder:scaffold:imports

	googlecloudmetadata "cloud.google.com/go/compute/metadata"

	_ "net/http/pprof"
)

var (
	scheme                = runtime.NewScheme()
	setupLog              = ctrl.Log.WithName("setup")
	defaultCollectorImage = "registry.odigos.io/odigos-collector"
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
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

	odigosVersion := os.Getenv(consts.OdigosVersionEnvVarName)
	if odigosVersion == "" {
		flag.StringVar(&odigosVersion, "version", "", "for development purposes only")
	}
	err := feature.Setup()
	if err != nil {
		setupLog.Error(err, "unable to get setup feature k8s detection")
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

	ctx := ctrl.SetupSignalHandler()
	go common.StartPprofServer(ctx, setupLog)

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
				&odigosv1.CollectorsGroup{}: {
					Field: nsSelector,
				},
				&odigosv1.Destination{}: {
					Field: nsSelector,
				},
				&odigosv1.Processor{}: {
					Field: nsSelector,
				},
				&apiactions.AddClusterInfo{}: {
					Field: nsSelector,
				},
				&apiactions.DeleteAttribute{}: {
					Field: nsSelector,
				},
				&apiactions.ErrorSampler{}: {
					Field: nsSelector,
				},
				&apiactions.LatencySampler{}: {
					Field: nsSelector,
				},
				&apiactions.SpanAttributeSampler{}: {
					Field: nsSelector,
				},
				&apiactions.ServiceNameSampler{}: {
					Field: nsSelector,
				},
				&apiactions.PiiMasking{}: {
					Field: nsSelector,
				},
				&apiactions.ProbabilisticSampler{}: {
					Field: nsSelector,
				},
				&apiactions.RenameAttribute{}: {
					Field: nsSelector,
				},
				&apiactions.K8sAttributesResolver{}: {
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

	collectorImage := defaultCollectorImage
	if collectorImageEnv, ok := os.LookupEnv("ODIGOS_COLLECTOR_IMAGE"); ok {
		collectorImage = collectorImageEnv
	}

	// TODO: this should be removed once the hpa logic uses the feature package for its checks
	k8sVersion := feature.K8sVersion()
	// this is a workaround because the GKE detector does not respect the timeout configuration for the resource detection processor.
	// it could lead to long initialization times for the data-collection,
	// as a workaround we try to understand here if we're on GKE with a timeout of 2 seconds.
	// TODO: remove this once https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/issues/1026 is resolved.
	// DO NOT ADD SIMILAR FUNCTIONS FOR OTHER PLATFORMS
	onGKE := isRunningOnGKE(ctx)
	if onGKE {
		setupLog.Info("Running on GKE")
	}

	commonconfig.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion:     k8sVersion,
		CollectorImage: collectorImage,
		OnGKE:          onGKE,
	}

	if err = (&controllers.DestinationReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Destination")
		os.Exit(1)
	}

	if err = (&controllers.ProcessorReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Processor")
		os.Exit(1)
	}
	if err = (&controllers.CollectorsGroupReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "CollectorsGroup")
		os.Exit(1)
	}
	if err = (&controllers.InstrumentationConfigReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InstrumentationConfig")
		os.Exit(1)
	}
	if err = (&controllers.SecretReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
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
	if err = (&controllers.SourceReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Source")
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
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// based on https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/blob/19c4db6ea12211308fbd2cba12cc8665a5b7c890/detectors/gcp/gke.go#L34
func isRunningOnGKE(ctx context.Context) bool {
	c := googlecloudmetadata.NewClient(nil)
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.InstanceAttributeValueWithContext(ctx, "cluster-location")
	return err == nil
}
