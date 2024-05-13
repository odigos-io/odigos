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

	if becameEnabled {
		return true
	}

	replicasBecameAvailable := didReplicasBecomeAvailable(e.ObjectOld, e.ObjectNew)
	if replicasBecameAvailable {
		return true
	}

	// The language detection process currently does 2 things:
	// 1. Detect the language of each container in the workload
	// 2. Detect the actual value of relevant environment variables for each container.
	//
	// thus, we need to re-run language detection if something that might affect
	// any of these 2 things has changed.
	//
	// currently, we only check if the enabled label has changed, or the pod become available,
	// but other events that might affect the language detection are not checked.
	// for example: if the container array changed, if an env var was added/removed, if the image was changed, etc.
	// we might need to add these checks in the future.
	// notice that the change alone is not enough - after the change, the workload running pods still
	// run an old manifest. we should re-calculate the runtime details only with up-to-date running pods.

	return false
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

// language detection relies on the fact that the workload has available replicas.
// if we did not have available replicas before, and now we do, we need to re-run language detection
func didReplicasBecomeAvailable(old client.Object, new client.Object) bool {

	switch new.(type) {
	case *appsv1.Deployment:
		hadAvailableReplicas := isDeploymentAvailableReplicas(new.(*appsv1.Deployment))
		hasAvailableReplicas := isDeploymentAvailableReplicas(old.(*appsv1.Deployment))
		return !hadAvailableReplicas && hasAvailableReplicas
	case *appsv1.DaemonSet:
		hadAvailableReplicas := isDaemonsetAvailableReplicas(new.(*appsv1.DaemonSet))
		hasAvailableReplicas := isDaemonsetAvailableReplicas(old.(*appsv1.DaemonSet))
		return !hadAvailableReplicas && hasAvailableReplicas
	case *appsv1.StatefulSet:
		hadAvailableReplicas := isStatefulsetAvailableReplicas(new.(*appsv1.StatefulSet))
		hasAvailableReplicas := isStatefulsetAvailableReplicas(old.(*appsv1.StatefulSet))
		return !hadAvailableReplicas && hasAvailableReplicas
	}

	return false
}
