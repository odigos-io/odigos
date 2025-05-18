package nodecollector

import (
	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type receiverSignalsChangedPredicate struct {
}

func (o receiverSignalsChangedPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}

	return true
}

func (i receiverSignalsChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil || e.ObjectOld == nil {
		return false
	}

	oldCollectorGroup, ok := e.ObjectOld.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}
	newCollectorGroup, ok := e.ObjectNew.(*odigosv1.CollectorsGroup)
	if !ok {
		return false
	}

	// check if the receiver signals array has changed (len or content)
	if len(oldCollectorGroup.Status.ReceiverSignals) != len(newCollectorGroup.Status.ReceiverSignals) {
		return true
	}
	for i := 0; i < len(oldCollectorGroup.Status.ReceiverSignals); i++ {
		if oldCollectorGroup.Status.ReceiverSignals[i] != newCollectorGroup.Status.ReceiverSignals[i] {
			return true
		}
	}

	return false
}

func (i receiverSignalsChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i receiverSignalsChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &receiverSignalsChangedPredicate{}

func SetupWithManager(mgr ctrl.Manager, imagePullSecrets []string, odigosVersion string) error {
	err := builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-collectorsgroup").
		For(&odigosv1.CollectorsGroup{}).
		Owns(&appsv1.DaemonSet{}). // in case the node-collector ds is deleted or modified for any reason, this will reconcile and recreate it
		Owns(&corev1.ConfigMap{}). // in case the configmap of the node-collector is deleted or modified for any reason, this will reconcile and recreate it
		// we assume everything in the collectorsgroup spec is the configuration for the collectors to generate.
		// thus, we need to monitor any change to the spec which is what the generation field is for.
		WithEventFilter(
			predicate.Or(
				predicate.And(&odigospredicate.OdigosCollectorsGroupNodePredicate, &predicate.GenerationChangedPredicate{}),
				predicate.And(&odigospredicate.OdigosCollectorsGroupClusterPredicate, &receiverSignalsChangedPredicate{}),
			)).
		Complete(&CollectorsGroupReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			ImagePullSecrets: imagePullSecrets,
			OdigosVersion:    odigosVersion,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-instrumentationconfig").
		For(&odigosv1.InstrumentationConfig{}).
		// this controller only cares about the instrumented application existence.
		// when it is created or removed, the node collector config map needs to be updated to scrape logs for it's pods.
		WithEventFilter(&odigospredicate.ExistencePredicate{}).
		Complete(&InstrumentationConfigReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			ImagePullSecrets: imagePullSecrets,
			OdigosVersion:    odigosVersion,
		})

	err = builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-processor").
		For(&odigosv1.Processor{}).
		// auto scaler only cares about the spec of each processor.
		// filter out events on resource status and metadata changes.
		WithEventFilter(&predicate.GenerationChangedPredicate{}).
		Complete(&ProcessorReconciler{
			Client:           mgr.GetClient(),
			Scheme:           mgr.GetScheme(),
			ImagePullSecrets: imagePullSecrets,
			OdigosVersion:    odigosVersion,
		})
	if err != nil {
		return err
	}

	err = builder.
		ControllerManagedBy(mgr).
		Named("nodecollector-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(&odigospredicate.NodeCollectorsDaemonSetPredicate).
		Complete(&NodeCollectorDaemonSetReconciler{
			Client: mgr.GetClient(),
		})
	if err != nil {
		return err
	}

	return nil
}
