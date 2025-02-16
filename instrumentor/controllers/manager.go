package controllers

import (
	"fmt"
	"os"
	"time"

	"github.com/go-logr/logr"

	"github.com/odigos-io/odigos/instrumentor/controllers/agentenabled"
	"github.com/odigos-io/odigos/instrumentor/controllers/deleteinstrumentationconfig"
	"github.com/odigos-io/odigos/instrumentor/controllers/instrumentationconfig"
	"github.com/odigos-io/odigos/instrumentor/controllers/startlangdetection"
	"github.com/odigos-io/odigos/instrumentor/controllers/workloadmigrations"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	corev1 "k8s.io/api/core/v1"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(odigosv1.AddToScheme(scheme))
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
			BindAddress: opts.MetricsServerBindAddress,
		},
		HealthProbeBindAddress: opts.HealthProbeBindAddress,
		LeaderElection:         opts.EnableLeaderElection,
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

	return ctrl.NewManager(ctrl.GetConfigOrDie(), mgrOptions)
}

func durationPointer(d time.Duration) *time.Duration {
	return &d
}

func SetupWithManager(mgr manager.Manager) error {
	err := agentenabled.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to create controller for agent enabled: %w", err)
	}

	err = deleteinstrumentationconfig.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to create controller for delete instrumentation config: %w", err)
	}

	err = startlangdetection.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to create controller for start language detection: %w", err)
	}

	err = instrumentationconfig.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to create controller for instrumentation config: %w", err)
	}

	err = workloadmigrations.SetupWithManager(mgr)
	if err != nil {
		return fmt.Errorf("failed to create controller for workload migrations: %w", err)
	}

	// TODO: this webhook is relevant for both the startlangdetection and deleteinstrumentationconfig controllers
	// we are planning to combine these controllers into a single controller in the future,
	// so we can move this webhook to that combined controller
	err = builder.
		WebhookManagedBy(mgr).
		For(&odigosv1.Source{}).
		WithDefaulter(&SourcesDefaulter{
			Client: mgr.GetClient(),
		}).
		WithValidator(&SourcesValidator{
			Client: mgr.GetClient(),
		}).
		Complete()
	if err != nil {
		return err
	}

	return nil
}
