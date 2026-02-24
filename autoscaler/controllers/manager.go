package controllers

import (
	"fmt"
	"time"

	"github.com/go-logr/logr"

	apiactions "github.com/odigos-io/odigos/api/actions/v1alpha1"
	"github.com/odigos-io/odigos/autoscaler/controllers/actions"
	"github.com/odigos-io/odigos/autoscaler/controllers/clustercollector"
	"github.com/odigos-io/odigos/autoscaler/controllers/metricshandler"
	"github.com/odigos-io/odigos/autoscaler/controllers/nodecollector"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
	utilruntime.Must(apiactions.AddToScheme(scheme))
	utilruntime.Must(apiregv1.AddToScheme(scheme))
}

type KubeManagerOptions struct {
	Logger                   logr.Logger
	MetricsServerBindAddress string
	HealthProbeBindAddress   string
	EnableLeaderElection     bool
}

func CreateManager(opts KubeManagerOptions) (ctrl.Manager, error) {
	ctrl.SetLogger(opts.Logger)

	odigosNs := env.GetCurrentNamespace()
	nsSelector := client.InNamespace(odigosNs).AsSelector()
	clusterCollectorLabelSelector := labels.Set(clustercollector.ClusterCollectorGateway).AsSelector()

	mgrOptions := ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress: opts.MetricsServerBindAddress,
		},
		HealthProbeBindAddress: opts.HealthProbeBindAddress,
		LeaderElection:         opts.EnableLeaderElection,
		LeaderElectionID:       "f681cfed.odigos.io",
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
		Cache: cache.Options{
			DefaultTransform: cache.TransformStripManagedFields(),
			ByObject: map[client.Object]cache.ByObject{
				&appsv1.Deployment{}: {
					Field: nsSelector,
				},
				&corev1.Service{}: {
					Label: clusterCollectorLabelSelector,
					Field: nsSelector,
				},
				&corev1.Pod{}: {
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
				&odigosv1.Action{}: {
					Field: nsSelector,
				},
				&odigosv1.Sampling{}: {
					Field: nsSelector,
				},
				&odigosv1.InstrumentationConfig{}: {},
			},
		},
	}

	return ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
}

func durationPointer(d time.Duration) *time.Duration {
	return &d
}

func SetupWithManager(mgr manager.Manager, odigosVersion string) error {
	err := nodecollector.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to create controller for node collector: %w", err)
	}

	err = clustercollector.SetupWithManager(mgr, odigosVersion)
	if err != nil {
		return fmt.Errorf("failed to create controller for cluster collector: %w", err)
	}

	if err = actions.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create controller for actions: %w", err)
	}

	if err = metricshandler.SetupWithManager(mgr); err != nil {
		return fmt.Errorf("failed to create controller for metrics handler: %w", err)
	}

	return nil
}

func RegisterWebhooks(mgr manager.Manager) error {
	return actions.RegisterWebhooks(mgr)
}
