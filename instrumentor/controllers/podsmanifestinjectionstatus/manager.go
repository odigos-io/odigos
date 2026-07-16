package podsmanifestinjectionstatus

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

type InstrumentationConfigPodsManifestInjectionPredicate struct{}

func (o InstrumentationConfigPodsManifestInjectionPredicate) Create(e event.CreateEvent) bool {

	// at creation time, we need to fill the current pods injection status in the ic.
	// if instrumentor was down or restarting, we also need to sync the pods injection number
	// for any changes not being picked up while the controller was not able to process events.
	return true
}

func (o InstrumentationConfigPodsManifestInjectionPredicate) Update(e event.UpdateEvent) bool {
	old, oldOk := e.ObjectOld.(*odigosv1.InstrumentationConfig)
	new, newOk := e.ObjectNew.(*odigosv1.InstrumentationConfig)

	if !oldOk || !newOk {
		return false
	}

	// pods injection count uses the agents meta hash, and when it changes, we need to re-compute the couters to have them correct.
	if old.Spec.AgentsMetaHash != new.Spec.AgentsMetaHash {
		return true
	}

	// rollout progress / queue state affects which PodsManifestInjection reason we report
	if old.Status.WorkloadRolloutHash != new.Status.WorkloadRolloutHash {
		return true
	}
	oldRollout := meta.FindStatusCondition(old.Status.Conditions, odigosv1.WorkloadRolloutStatusConditionType)
	newRollout := meta.FindStatusCondition(new.Status.Conditions, odigosv1.WorkloadRolloutStatusConditionType)
	if (oldRollout == nil) != (newRollout == nil) {
		return true
	}
	if oldRollout != nil && newRollout != nil &&
		(oldRollout.Reason != newRollout.Reason || oldRollout.Status != newRollout.Status) {
		return true
	}

	return false
}

func (o InstrumentationConfigPodsManifestInjectionPredicate) Delete(e event.DeleteEvent) bool {
	// the status is written to the ic, so if it's deleted, we have nothing to do.
	return false
}

func (o InstrumentationConfigPodsManifestInjectionPredicate) Generic(e event.GenericEvent) bool {
	return true
}

func SetupWithManager(mgr ctrl.Manager) error {

	podsTracker := NewPodsTracker()

	err := builder.
		ControllerManagedBy(mgr).
		Named("podsmanifestinjection-pods").
		For(&corev1.Pod{}).
		WithEventFilter(odigospredicate.ExistencePredicate{}).
		Complete(
			&PodsController{
				Client:      mgr.GetClient(),
				PodsTracker: podsTracker,
			},
		)
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("podsmanifestinjection-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		WithEventFilter(&InstrumentationConfigPodsManifestInjectionPredicate{}).
		Complete(&InstrumentationConfigController{
			Client:      mgr.GetClient(),
			PodsTracker: podsTracker,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("podsmanifestinjection-effectiveconfig").
		For(&corev1.ConfigMap{}).
		WithEventFilter(odigospredicate.OdigosEffectiveConfigMapPredicate).
		Complete(&EffectiveConfigReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
