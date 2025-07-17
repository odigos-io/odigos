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
	"time"

	"github.com/go-logr/zapr"
	bridge "github.com/odigos-io/opentelemetry-zap-bridge"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/configmaps"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/dynamic"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	ctrlzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/odigos-io/odigos/scheduler/clusterinfo"
	"github.com/odigos-io/odigos/scheduler/controllers/clustercollectorsgroup"
	"github.com/odigos-io/odigos/scheduler/controllers/nodecollectorsgroup"
	"github.com/odigos-io/odigos/scheduler/controllers/odigosconfiguration"
	"github.com/odigos-io/odigos/scheduler/controllers/odigospro"
	//+kubebuilder:scaffold:imports
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
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
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
	tier := env.GetOdigosTierFromEnv()
	odigosVersion := os.Getenv(consts.OdigosVersionEnvVarName)

	nsSelector := client.InNamespace(odigosNs).AsSelector()

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: metricsAddr,
		},
		Cache: cache.Options{
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				&corev1.ConfigMap{}: {
					Field: nsSelector,
				},
				&odigosv1.CollectorsGroup{}: {
					Field: nsSelector,
				},
				&odigosv1.Processor{}: {
					Field: nsSelector,
				},
				&odigosv1.InstrumentationRule{}: {
					Field: nsSelector,
				},
				&corev1.Secret{}: {
					Field: nsSelector,
				},
			},
		},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "ce024640.odigos.io",
		/*
			Leader Election Parameters:

			LeaseDuration (30s):
			- Maximum time a pod can remain the leader after its last successful renewal.
			- If the leader pod dies, failover can take up to the LeaseDuration from the last renewal.
			  The actual failover time depends on how recently the leader renewed the lease.
			- Controls when the lease is fully expired and failover can occur.

			RenewDeadline (20s):
			- The maximum time the leader pod has to successfully renew its lease before it is
			  considered unhealthy. Relevant only while the leader is alive and renewing.
			- Controls how long the current leader will keep retrying to refresh the lease.

			RetryPeriod (5s):
			- How often non-leader pods check and attempt to acquire leadership when the lease is available.
			- Lower value means faster failover but adds more load on the Kubernetes API server.

			Relationship:
			- RetryPeriod < RenewDeadline < LeaseDuration
			- This ensures proper failover timing and system stability.

			Setting the leader election params to 30s/20s/5s should provide a good balance between stability and quick failover.
		*/
		LeaseDuration:                 durationPointer(30 * time.Second),
		RenewDeadline:                 durationPointer(20 * time.Second),
		RetryPeriod:                   durationPointer(5 * time.Second),
		LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// create dynamic k8s client to apply profile manifests
	dyanmicClient, err := dynamic.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create dynamic client")
		os.Exit(1)
	}

	clientset, err := kubernetes.NewForConfig(mgr.GetConfig())
	if err != nil {
		setupLog.Error(err, "unable to create kubernetes client")
		os.Exit(1)
	}
	err = clusterinfo.RecordClusterInfo(context.Background(), clientset, odigosNs)
	if err != nil {
		setupLog.Error(err, "unable to record cluster info, skipping")
	}

	err = clustercollectorsgroup.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controllers for cluster collectors group")
		os.Exit(1)
	}
	err = nodecollectorsgroup.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controllers for node collectors group")
		os.Exit(1)
	}
	err = odigosconfiguration.SetupWithManager(mgr, tier, odigosVersion, dyanmicClient)
	if err != nil {
		setupLog.Error(err, "unable to create controllers for odigos configuration")
		os.Exit(1)
	}
	err = odigospro.SetupWithManager(mgr)
	if err != nil {
		setupLog.Error(err, "unable to create controller for odigos pro")
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

	// remove the legacy configmap if it exists
	mgr.Add(&configmaps.ConfigMapDeleteMigration{Client: mgr.GetClient(), Logger: setupLog, ConfigMap: types.NamespacedName{
		Namespace: env.GetCurrentNamespace(),
		Name:      consts.OdigosLegacyConfigName,
	}})

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func durationPointer(d time.Duration) *time.Duration {
	return &d
}
