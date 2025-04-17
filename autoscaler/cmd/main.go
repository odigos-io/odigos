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
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiactions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"github.com/odigos-io/odigos/autoscaler/controllers"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"

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
	var imagePullSecretsString string
	var imagePullSecrets []string

	managerOptions := controllers.KubeManagerOptions{}

	flag.StringVar(&managerOptions.MetricsServerBindAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&managerOptions.HealthProbeBindAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&managerOptions.EnableLeaderElection, "leader-elect", false,
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

	mgr, err := controllers.CreateManager(managerOptions)
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

	// wire up the controllers and webhooks
	err = controllers.SetupWithManager(mgr, imagePullSecrets, odigosVersion)
	if err != nil {
		setupLog.Error(err, "unable to create odigos controllers")
		os.Exit(1)
	}

	if err = actions.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create odigos actions controllers")
		os.Exit(1)
	}

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
