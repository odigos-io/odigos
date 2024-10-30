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

	"github.com/odigos-io/odigos/k8sutils/pkg/env"

	"github.com/odigos-io/odigos/instrumentor/controllers/instrumentationconfig"
	"github.com/odigos-io/odigos/instrumentor/controllers/startlangdetection"
	"github.com/odigos-io/odigos/instrumentor/sdks"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	corev1 "k8s.io/api/core/v1"

	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/labels"

	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"

	v1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"

	"github.com/odigos-io/odigos/instrumentor/controllers/deleteinstrumentedapplication"
	"github.com/odigos-io/odigos/instrumentor/controllers/instrumentationdevice"
	"github.com/odigos-io/odigos/instrumentor/report"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	//+kubebuilder:scaffold:imports

	_ "net/http/pprof"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(v1.AddToScheme(scheme))
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

	mgrOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "201bdfa0.odigos.io",
		Cache: cache.Options{
			DefaultTransform: cache.TransformStripManagedFields(),
			// Store minimum amount of data for every object type.
			// Currently, instrumentor only need the labels and the .spec.template.spec field of the workloads.
			ByObject: map[client.Object]cache.ByObject{
				&appsv1.Deployment{}: {
					Transform: func(obj interface{}) (interface{}, error) {
						deployment := obj.(*appsv1.Deployment)
						newDep := &appsv1.Deployment{
							TypeMeta: deployment.TypeMeta,
							ObjectMeta: metav1.ObjectMeta{
								Name:        deployment.Name,
								Namespace:   deployment.Namespace,
								Labels:      deployment.Labels,
								Annotations: deployment.Annotations,
								UID:         deployment.UID,
							},
							Status: deployment.Status,
							Spec: appsv1.DeploymentSpec{
								Template: corev1.PodTemplateSpec{
									ObjectMeta: metav1.ObjectMeta{
										Labels: deployment.Spec.Template.Labels,
									},
								},
							},
						}
						newDep.Spec.Template.Spec = deployment.Spec.Template.Spec
						return newDep, nil
					},
				},
				&appsv1.StatefulSet{}: {
					Transform: func(obj interface{}) (interface{}, error) {
						ss := obj.(*appsv1.StatefulSet)
						newSs := &appsv1.StatefulSet{
							TypeMeta: ss.TypeMeta,
							ObjectMeta: metav1.ObjectMeta{
								Name:        ss.Name,
								Namespace:   ss.Namespace,
								Labels:      ss.Labels,
								Annotations: ss.Annotations,
								UID:         ss.UID,
							},
							Status: ss.Status,
						}

						newSs.Spec.Template.Spec = ss.Spec.Template.Spec
						return newSs, nil
					},
				},
				&appsv1.DaemonSet{}: {
					Transform: func(obj interface{}) (interface{}, error) {
						ds := obj.(*appsv1.DaemonSet)
						newDs := &appsv1.DaemonSet{
							TypeMeta: ds.TypeMeta,
							ObjectMeta: metav1.ObjectMeta{
								Name:        ds.Name,
								Namespace:   ds.Namespace,
								Labels:      ds.Labels,
								Annotations: ds.Annotations,
								UID:         ds.UID,
							},
							Status: ds.Status,
						}

						newDs.Spec.Template.Spec = ds.Spec.Template.Spec
						return newDs, nil
					},
				},
				&corev1.Namespace{}: {
					Label: labels.Set{consts.OdigosInstrumentationLabel: consts.InstrumentationEnabled}.AsSelector(),
				},
				&corev1.ConfigMap{}: {
					Field: client.InNamespace(env.GetCurrentNamespace()).AsSelector(),
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

	err = sdks.SetDefaultSDKs(ctx)

	if err != nil {
		setupLog.Error(err, "Failed to set default SDKs")
		os.Exit(-1)
	}

	err = instrumentationdevice.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		os.Exit(1)
	}

	err = deleteinstrumentedapplication.SetupWithManager(mgr)
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

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	go common.StartPprofServer(setupLog)

	if !telemetryDisabled {
		go report.Start(mgr.GetClient())
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
