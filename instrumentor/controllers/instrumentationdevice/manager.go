package instrumentationdevice

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func countOdigosResources(resources corev1.ResourceList) int {
	numOdigosResources := 0
	for resourceName := range resources {
		if common.IsResourceNameOdigosInstrumentation(resourceName.String()) {
			numOdigosResources = numOdigosResources + 1
		}
	}
	return numOdigosResources
}

type workloadPodTemplatePredicate struct {
	predicate.Funcs
}

func (w workloadPodTemplatePredicate) Create(e event.CreateEvent) bool {
	return false
}

func (w workloadPodTemplatePredicate) Update(e event.UpdateEvent) bool {

	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldPodSpec, err := getPodSpecFromObject(e.ObjectOld)
	if err != nil {
		return false
	}
	newPodSpec, err := getPodSpecFromObject(e.ObjectNew)
	if err != nil {
		return false
	}

	// only handle workloads if any env changed
	if len(oldPodSpec.Spec.Containers) != len(newPodSpec.Spec.Containers) {
		return true
	}
	for i := range oldPodSpec.Spec.Containers {
		if len(oldPodSpec.Spec.Containers[i].Env) != len(newPodSpec.Spec.Containers[i].Env) {
			return true
		}
		for j := range oldPodSpec.Spec.Containers[i].Env {
			prevEnv := &newPodSpec.Spec.Containers[i].Env[j]
			newEnv := &oldPodSpec.Spec.Containers[i].Env[j]
			if prevEnv.Name != newEnv.Name || prevEnv.Value != newEnv.Value {
				return true
			}
		}

		// user might apply a change to workload which will overwrite odigos injected resources
		prevNumOdigosResources := countOdigosResources(oldPodSpec.Spec.Containers[i].Resources.Limits)
		newNumOdigosResources := countOdigosResources(newPodSpec.Spec.Containers[i].Resources.Limits)
		if prevNumOdigosResources != newNumOdigosResources {
			return true
		}
	}

	return false
}

func (w workloadPodTemplatePredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (w workloadPodTemplatePredicate) Generic(e event.GenericEvent) bool {
	return false
}

func SetupWithManager(mgr ctrl.Manager) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-collectorsgroup").
		For(&odigosv1.CollectorsGroup{}).
		Complete(&CollectorsGroupReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-instrumentedapplication").
		For(&odigosv1.InstrumentedApplication{}).
		Complete(&InstrumentedApplicationReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-configmaps").
		For(&corev1.ConfigMap{}).
		WithEventFilter(&utils.OnlyUpdatesPredicate{}).
		Complete(&OdigosConfigReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(workloadPodTemplatePredicate{}).
		Complete(&DeploymentReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(workloadPodTemplatePredicate{}).
		Complete(&DaemonSetReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		WithEventFilter(workloadPodTemplatePredicate{}).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-instrumentationrules").
		For(&odigosv1.InstrumentationRule{}).
		WithEventFilter(&utils.OtelSdkInstrumentationRulePredicate{}).
		Complete(&InstrumentationRuleReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	err = builder.
		WebhookManagedBy(mgr).
		For(&corev1.Pod{}).
		WithDefaulter(&PodsWebhook{
			Client: mgr.GetClient(),
		}).
		Complete()
	if err != nil {
		return err
	}

	return nil
}
