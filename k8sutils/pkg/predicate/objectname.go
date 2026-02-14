package predicate

import (
	"sigs.k8s.io/controller-runtime/pkg/event"
	cr_predicate "sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/common/consts"
)

type ObjectNamePredicate struct {
	AllowedObjectName string
}

func (i ObjectNamePredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}
	return e.Object.GetName() == i.AllowedObjectName
}

func (i ObjectNamePredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil || e.ObjectOld == nil {
		return false
	}
	return e.ObjectNew.GetName() == i.AllowedObjectName
}

func (i ObjectNamePredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}
	return e.Object.GetName() == i.AllowedObjectName
}

func (i ObjectNamePredicate) Generic(e event.GenericEvent) bool {
	if e.Object == nil {
		return false
	}
	return e.Object.GetName() == i.AllowedObjectName
}

var _ cr_predicate.Predicate = &ObjectNamePredicate{}

// This predicate will only allow config map events on the "odigos-configuration" object,
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

var OdigosEffectiveConfigMapPredicate = ObjectNamePredicate{
	AllowedObjectName: consts.OdigosEffectiveConfigName,
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
var OdigosCollectorsGroupNodePredicate = ObjectNamePredicate{
	AllowedObjectName: k8sconsts.OdigosNodeCollectorCollectorGroupName,
}

// use this event filter to reconcile only collectors group events for cluster collectors group objects
// this is useful if you reconcile only depends on changes from the cluster collectors group and should not react to node collectors group changes
// example usage:
// import odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
// ...
//
//	err = ctrl.NewControllerManagedBy(mgr).
//		For(&odigosv1.CollectorsGroup{}).
//		WithEventFilter(&odigospredicates.OdigosCollectorsGroupClusterPredicate).
//		Complete(r)
var OdigosCollectorsGroupClusterPredicate = ObjectNamePredicate{
	AllowedObjectName: k8sconsts.OdigosClusterCollectorCollectorGroupName,
}

var OdigosProSecretPredicate = ObjectNamePredicate{
	AllowedObjectName: k8sconsts.OdigosProSecretName,
}
var OdigosDeploymentConfigMapPredicate = ObjectNamePredicate{
	AllowedObjectName: k8sconsts.OdigosDeploymentConfigMapName,
}

// OdigosRemoteConfigMapPredicate filters events for the odigos-remote-config ConfigMap
// which contains configuration managed by the central-backend.
var OdigosRemoteConfigMapPredicate = ObjectNamePredicate{
	AllowedObjectName: consts.OdigosRemoteConfigName,
}
