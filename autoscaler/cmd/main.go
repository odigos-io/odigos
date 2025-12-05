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
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/certs"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"github.com/odigos-io/odigos/k8sutils/pkg/feature"
	"github.com/open-policy-agent/cert-controller/pkg/rotator"
	"golang.org/x/sync/errgroup"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiactions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"github.com/odigos-io/odigos/autoscaler/controllers"
	commonconfig "github.com/odigos-io/odigos/autoscaler/controllers/common"
	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/metricshandler"
	"github.com/odigos-io/odigos/autoscaler/controllers/nodecollector"

	//+kubebuilder:scaffold:imports

	googlecloudmetadata "cloud.google.com/go/compute/metadata"

	"net/http"
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
	utilruntime.Must(apiregv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	managerOptions := controllers.KubeManagerOptions{}

	flag.StringVar(&managerOptions.MetricsServerBindAddress, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&managerOptions.HealthProbeBindAddress, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&managerOptions.EnableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

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

	zapLogger := ctrlzap.NewRaw(ctrlzap.UseFlagOptions(&opts))
	zapLogger = bridge.AttachToZapLogger(zapLogger)
	logger := zapr.NewLogger(zapLogger)
	ctrl.SetLogger(logger)

	if odigosVersion == "" {
		setupLog.Error(nil, "ODIGOS_VERSION environment variable is not set and version flag is not provided")
		os.Exit(1)
	}

	ctx := ctrl.SetupSignalHandler()
	go common.StartPprofServer(ctx, setupLog, int(k8sconsts.DefaultPprofEndpointPort))

	setupLog.Info("Starting odigos autoscaler", "version", odigosVersion)

	mgr, err := controllers.CreateManager(managerOptions)
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// remove the deprecated webhook secret if it exists
	mgr.Add(&certs.SecretDeleteMigration{Client: mgr.GetClient(), Logger: logger, Secret: types.NamespacedName{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.DeprecatedAutoscalerWebhookSecretName,
	}})

	// remove the data collection daemonset if exists because it is part of the odiglet pod now.
	// TODO: once we're done with the migration, we can remove this.
	mgr.Add(&nodecollector.DataCollectionDSMigration{Client: mgr.GetClient(), Logger: logger, DataCollectionDaemonSet: types.NamespacedName{
		Namespace: env.GetCurrentNamespace(),
		Name:      k8sconsts.OdigosNodeCollectorDaemonSetName,
	}})

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

	rotatorSetupFinished := make(chan struct{})
	err = rotator.AddRotator(mgr, &rotator.CertRotator{
		SecretKey: types.NamespacedName{
			Namespace: env.GetCurrentNamespace(),
			Name:      k8sconsts.AutoscalerWebhookSecretName,
		},
		CertDir: filepath.Join(os.TempDir(), "k8s-webhook-server", "serving-certs"),
		IsReady: rotatorSetupFinished,
		CAName:  k8sconsts.AutoscalerCAName,
		Webhooks: []rotator.WebhookInfo{
			{Name: k8sconsts.AutoscalerActionValidatingWebhookName, Type: rotator.Validating},
		},
		DNSName: "serving-cert",
		ExtraDNSNames: []string{
			fmt.Sprintf("%s.%s.svc", k8sconsts.AutoScalerWebhookServiceName, env.GetCurrentNamespace()),
			fmt.Sprintf("%s.%s.svc.cluster.local", k8sconsts.AutoScalerWebhookServiceName, env.GetCurrentNamespace()),
		},
		EnableReadinessCheck: true,

		// marking the controller as the owner of the webhooks config updated fields.
		// this avoid CI/CD systems overwriting the managed fields.
		FieldOwner: k8sconsts.AutoScalerWebhookFieldOwner,

		// these are the defaults, but we set them explicitly for clarity
		CaCertDuration:         10 * 365 * 24 * time.Hour, // 10 years
		ServerCertDuration:     1 * 365 * 24 * time.Hour,  // 1 year
		RotationCheckFrequency: 12 * time.Hour,            // 12 hours
		LookaheadInterval:      90 * 24 * time.Hour,       // 90 days
	})
	if err != nil {
		setupLog.Error(err, "unable to add cert rotator")
		os.Exit(1)
	}

	commonconfig.ControllerConfig = &controllerconfig.ControllerConfig{
		K8sVersion:     k8sVersion,
		CollectorImage: collectorImage,
		OnGKE:          onGKE,
	}

	// wire up the controllers
	err = controllers.SetupWithManager(mgr, odigosVersion)
	if err != nil {
		setupLog.Error(err, "unable to create odigos controllers")
		os.Exit(1)
	}

	webhooksRegistered := atomic.Bool{}
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		return mgr.GetWebhookServer().StartedChecker()(req)
	}); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", func(req *http.Request) error {
		if !webhooksRegistered.Load() {
			return errors.New("webhooks not registered yet")
		}
		return nil
	}); err != nil {
		setupLog.Error(err, "unable to set up cert rotator check")
		os.Exit(1)
	}

	g, groupCtx := errgroup.WithContext(ctx)
	// start kube manager
	g.Go(func() error {
		err := mgr.Start(groupCtx)
		if err != nil {
			setupLog.Error(err, "error starting kube manager")
		} else {
			setupLog.V(0).Info("Kube manager exited")
		}
		return err
	})

	// register webhooks after the certificate is ready
	g.Go(func() error {
		select {
		case <-rotatorSetupFinished:
		case <-groupCtx.Done():
			return nil
		}
		setupLog.V(0).Info("Cert rotator is ready")

		// Register admission webhooks
		err := controllers.RegisterWebhooks(mgr)
		if err != nil {
			return err
		}

		// Register Custom Metrics API
		// We use this to trigger the HPA of the gateway collector by aggregating the metrics
		// from all the gateway collector pods.
		if err := metricshandler.RegisterCustomMetricsAPI(mgr); err != nil {
			setupLog.Error(err, "failed to register custom metrics API")
		} else {
			setupLog.Info("Custom Metrics API registered successfully")
		}

		webhooksRegistered.Store(true)
		setupLog.V(0).Info("Webhooks registered")
		return nil
	})

	err = g.Wait()
	if err != nil {
		setupLog.Error(err, "autoscaler exited with error")
	} else {
		setupLog.V(0).Info("autoscaler exiting")
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
