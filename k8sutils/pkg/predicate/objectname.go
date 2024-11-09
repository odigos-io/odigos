package predicate

import (
	"github.com/odigos-io/odigos/common/consts"
	odigosk8sconsts "github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"
)

type ObjectNamePredicate struct {
	AllowedObjectName string
}

func (o ObjectNamePredicate) Create(e event.CreateEvent) bool {
	return e.Object.GetName() == o.AllowedObjectName
}

func (i ObjectNamePredicate) Update(e event.UpdateEvent) bool {
	return e.ObjectNew.GetName() == i.AllowedObjectName
}

func (i ObjectNamePredicate) Delete(e event.DeleteEvent) bool {
	return e.Object.GetName() == i.AllowedObjectName
}

func (i ObjectNamePredicate) Generic(e event.GenericEvent) bool {
	return e.Object.GetName() == i.AllowedObjectName
}

var _ cr_predicate.Predicate = &ObjectNamePredicate{}

// This predicate will only allow config map events on the "odigos-config" object,
// and will filter out events for possible other config maps which the reconciler should not handle.
// Example usage:
// import odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
// ...
//
//	err = ctrl.NewControllerManagedBy(mgr).
//		For(&corev1.ConfigMap{}).
//		WithEventFilter(&odigospredicates.OdigosConfigMapPredicate).
//		Complete(r)
var OdigosConfigMapPredicate = ObjectNamePredicate{
	AllowedObjectName: consts.OdigosConfigurationName,
}

// use this event filter to reconcile only collectors group events for node collectors group objects
// this is useful if you reconcile only depends on changes from the node collectors group and should not react to cluster collectors group changes
// example usage:
// import odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
// ...
//
//	err = ctrl.NewControllerManagedBy(mgr).
//		For(&odigosv1.CollectorsGroup{}).
//		WithEventFilter(&odigospredicates.OdigosCollectorsGroupCluster).
//		Complete(r)
var OdigosCollectorsGroupNode = ObjectNamePredicate{
	AllowedObjectName: odigosk8sconsts.OdigosNodeCollectorCollectorGroupName,
}

// use this event filter to reconcile only collectors group events for cluster collectors group objects
// this is useful if you reconcile only depends on changes from the cluster collectors group and should not react to node collectors group changes
// example usage:
// import odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
// ...
//
//	err = ctrl.NewControllerManagedBy(mgr).
//		For(&odigosv1.CollectorsGroup{}).
//		WithEventFilter(&odigospredicates.OdigosCollectorsGroupCluster).
//		Complete(r)
var OdigosCollectorsGroupCluster = ObjectNamePredicate{
	AllowedObjectName: odigosk8sconsts.OdigosClusterCollectorCollectorGroupName,
}
