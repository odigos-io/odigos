package instrumentationdevice

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common"
	"github.com/odigos-io/odigos/instrumentor/controllers/utils"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
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

type runtimeDetectionGeneration struct {
	predicate.Funcs
}

func (n runtimeDetectionGeneration) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	prevInstrumentationConfig, ok := e.ObjectOld.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}
	newInstrumentationConfig, ok := e.ObjectNew.(*odigosv1.InstrumentationConfig)
	if !ok {
		return false
	}

	return prevInstrumentationConfig.Status.ObservedWorkloadGeneration != newInstrumentationConfig.Status.ObservedWorkloadGeneration
}

type workloadPodTemplatePredicate struct {
	predicate.Funcs
}

func (w workloadPodTemplatePredicate) Create(e event.CreateEvent) bool {
	// when instrumentor restarts, this case will be triggered as workloads objects are being added to the cache.
	// in this case, we need to reconcile the workload, and guarantee that the device is injected or removed
	// based on the current state of the cluster.
	// if the instrumented application is deleted but the device is not cleaned,
	// the instrumented application controller will not be invoked after restart, which is why we need to handle this case here.
	return true
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
		WithEventFilter(predicate.And(&odigospredicate.OdigosCollectorsGroupNodePredicate, &odigospredicate.CgBecomesReadyPredicate{})).
		Complete(&CollectorsGroupReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentationdevice-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&runtimeDetectionGeneration{}).
		Complete(&InstrumentationConfigReconciler{
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
		WithDefaulter(&PodsWebhook{}).
		Complete()
	if err != nil {
		return err
	}

	return nil
}
