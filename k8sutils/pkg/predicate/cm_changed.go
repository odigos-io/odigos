package predicate

import (
	"maps"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// this event filter will only trigger reconciliation when the configmap data was changed.
// the reconciled type must be corev1.ConfigMap
// note: this preidcate currently only check the Data field of the ConfigMap (without BinaryData)
type ConfigMapDataChangedPredicate struct{}

func (o ConfigMapDataChangedPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i ConfigMapDataChangedPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldConfigMap, ok := e.ObjectOld.(*corev1.ConfigMap)
	if !ok {
		return false
	}
	newConfigMap, ok := e.ObjectNew.(*corev1.ConfigMap)
	if !ok {
		return false
	}

	return !maps.Equal(oldConfigMap.Data, newConfigMap.Data)
}

func (i ConfigMapDataChangedPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i ConfigMapDataChangedPredicate) Generic(e event.GenericEvent) bool {
	return false
}

var _ predicate.Predicate = &ConfigMapDataChangedPredicate{}
