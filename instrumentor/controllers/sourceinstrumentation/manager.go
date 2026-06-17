package sourceinstrumentation

import (
	openshiftappsv1 "github.com/openshift/api/apps/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/version"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	argorolloutsv1alpha1 "github.com/argoproj/argo-rollouts/pkg/apis/rollouts/v1alpha1"
	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/client"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
)

// TODO: deprecate this function and use k8sutils.IsResourceAvailable instead
// isDeploymentConfigAvailable checks if the DeploymentConfig resource is available in the cluster
// using the RESTMapper to avoid permission errors on non-OpenShift clusters
func isDeploymentConfigAvailable(mgr ctrl.Manager) bool {
	gvk := schema.GroupVersionKind{
		Group:   "apps.openshift.io",
		Version: "v1",
		Kind:    "DeploymentConfig",
	}

	// Try to get the REST mapping for DeploymentConfig
	// This will fail if the resource doesn't exist in the cluster
	_, err := mgr.GetRESTMapper().RESTMapping(gvk.GroupKind(), gvk.Version)
	return err == nil
}

func SetupWithManager(mgr ctrl.Manager, k8sVersion *version.Version) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-source").
		For(&v1alpha1.Source{}).
		Complete(&SourceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Workload reconcilers catch workloads created after their Source, and updates while the
	// workload's InstrumentationConfig is missing (for example after GitOps replace).
	workloadEventFilter := odigospredicate.WorkloadCreateOrMissingInstrumentationConfig(mgr.GetClient())

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(workloadEventFilter).
		Complete(&DeploymentReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(workloadEventFilter).
		Complete(&DaemonSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-statefulset").
		For(&appsv1.StatefulSet{}).
		WithEventFilter(workloadEventFilter).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-cronjob").
		For(&batchv1.CronJob{}).
		WithEventFilter(workloadEventFilter).
		Complete(&CronJobReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-namespace").
		For(&v1.Namespace{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&NamespaceReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("sourceinstrumentation-instrumentationconfig").
		For(&v1alpha1.InstrumentationConfig{}).
		WithEventFilter(&odigospredicate.CreationPredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Only register the DeploymentConfig controller if the resource is available (OpenShift clusters)
	// This avoids permission errors on non-OpenShift clusters where the resource doesn't exist
	if isDeploymentConfigAvailable(mgr) {
		err = builder.
			ControllerManagedBy(mgr).
			Named("sourceinstrumentation-deploymentconfig").
			For(&openshiftappsv1.DeploymentConfig{}).
			WithEventFilter(workloadEventFilter).
			Complete(&DeploymentConfigReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
			})
		if err != nil {
			return err
		}
	}

	// Only register the Rollout controller if the resource is available (Argo Rollouts installed on cluster)
	if k8sutils.IsResourceAvailable(mgr.GetRESTMapper(), k8sconsts.ArgoRolloutGVK) {
		err = builder.
			ControllerManagedBy(mgr).
			Named("sourceinstrumentation-rollout").
			For(&argorolloutsv1alpha1.Rollout{}).
			WithEventFilter(workloadEventFilter).
			Complete(&RolloutReconciler{
				Client: mgr.GetClient(),
				Scheme: mgr.GetScheme(),
			})
		if err != nil {
			return err
		}
	}

	return nil
}
