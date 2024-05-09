package runtime_details

import (
	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// this predicate is used for workload reconciler, and will only pass events
// where the workload is changed to odigos instrumentation enabled.
// This way, we don't need to run language detection downstream when unnecessary.
// This also helps in managing race conditions, where we might re-add runtime details
// which were just deleted by instrumentor controller and generate unnecessary noise
// in the k8s eventual consistency model.
type WorkloadEnabledPredicate struct {
	predicate.Funcs
}

func (i *WorkloadEnabledPredicate) Create(e event.CreateEvent) bool {
	enabled := isInstrumentationEnabled(e.Object)
	// only handle new workloads if they start with instrumentation enabled
	return enabled
}

func (i *WorkloadEnabledPredicate) Update(e event.UpdateEvent) bool {

	if e.ObjectOld == nil {
		return false
	}
	if e.ObjectNew == nil {
		return false
	}

	// only run runtime inspection if the workload was not instrumented before
	// and now it is.
	oldEnabled := isInstrumentationEnabled(e.ObjectOld)
	newEnabled := isInstrumentationEnabled(e.ObjectNew)
	becameEnabled := !oldEnabled && newEnabled

	switch e.ObjectNew.GetObjectKind().GroupVersionKind().Kind {
	case "Deployment":
		oldDeployment, oldOk := e.ObjectOld.(*appsv1.Deployment)
		newDeployment, newOk := e.ObjectNew.(*appsv1.Deployment)
		if oldOk && newOk {
			hadAvailableReplicas := isDeploymentAvailableReplicas(oldDeployment)
			hasAvailableReplicas := isDeploymentAvailableReplicas(newDeployment)
			replicasBecameAvailable := !hadAvailableReplicas && hasAvailableReplicas
			return becameEnabled || replicasBecameAvailable
		}
	case "DaemonSet":
		oldDaemonSet, oldOk := e.ObjectOld.(*appsv1.DaemonSet)
		newDaemonSet, newOk := e.ObjectNew.(*appsv1.DaemonSet)
		if oldOk && newOk {
			hadAvailableReplicas := isDaemonsetAvailableReplicas(oldDaemonSet)
			hasAvailableReplicas := isDaemonsetAvailableReplicas(newDaemonSet)
			replicasBecameAvailable := !hadAvailableReplicas && hasAvailableReplicas
			return becameEnabled || replicasBecameAvailable
		}
	case "StatefulSet":
		oldStatefulSet, oldOk := e.ObjectOld.(*appsv1.StatefulSet)
		newStatefulSet, newOk := e.ObjectNew.(*appsv1.StatefulSet)
		if oldOk && newOk {
			hadAvailableReplicas := isStatefulsetAvailableReplicas(oldStatefulSet)
			hasAvailableReplicas := isStatefulsetAvailableReplicas(newStatefulSet)
			replicasBecameAvailable := !hadAvailableReplicas && hasAvailableReplicas
			return becameEnabled || replicasBecameAvailable
		}
	}

	// for namespace events or if there was issue with type casting
	return becameEnabled
}

func (i *WorkloadEnabledPredicate) Delete(e event.DeleteEvent) bool {
	// no need to calculate runtime details for deleted workloads
	return false
}

func (i *WorkloadEnabledPredicate) Generic(e event.GenericEvent) bool {
	// not sure when exactly this would be called, but we don't need to handle it
	return false
}

func isInstrumentationEnabled(obj client.Object) bool {
	labels := obj.GetLabels()
	if labels == nil {
		return false
	}
	return labels[consts.OdigosInstrumentationLabel] == consts.InstrumentationEnabled
}

func isDeploymentAvailableReplicas(dep *appsv1.Deployment) bool {
	return dep.Status.AvailableReplicas > 0
}

func isDaemonsetAvailableReplicas(dep *appsv1.DaemonSet) bool {
	return dep.Status.NumberReady > 0
}

func isStatefulsetAvailableReplicas(dep *appsv1.StatefulSet) bool {
	return dep.Status.ReadyReplicas > 0
}
