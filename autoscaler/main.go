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
	"os"
	"strings"

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
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	apiactions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	observabilitycontrolplanev1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"

	"github.com/odigos-io/odigos/autoscaler/controllers"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	nameutils "github.com/odigos-io/odigos/autoscaler/utils"
	//+kubebuilder:scaffold:imports
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
	odigosVersion := os.Getenv("ODIGOS_VERSION")

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&imagePullSecretsString, "image-pull-secrets", "",
		"The image pull secrets to use for the collectors created by autoscaler")
	flag.StringVar(&nameutils.ImagePrefix, "image-prefix", "", "The image prefix to use for the collectors created by autoscaler")

	if odigosVersion == "" {
		flag.StringVar(&odigosVersion, "version", "", "for development purposes only")
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
	setupLog.Info("Starting odigos autoscaler", "version", odigosVersion)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:   enableLeaderElection,
		LeaderElectionID: "f681cfed.odigos.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
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
	if err = (&controllers.InstrumentedApplicationReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "InstrumentedApplication")
		os.Exit(1)
	}
	if err = (&controllers.OdigosConfigReconciler{
		Client:           mgr.GetClient(),
		Scheme:           mgr.GetScheme(),
		ImagePullSecrets: imagePullSecrets,
		OdigosVersion:    odigosVersion,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "OdigosConfig")
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
