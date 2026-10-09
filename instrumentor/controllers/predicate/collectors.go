package predicate

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func hasClusterCollectorLabel(obj metav1.Object) bool {
	return obj.GetLabels()[k8sconsts.OdigosCollectorRoleLabel] == string(k8sconsts.CollectorsRoleClusterGateway)
}

// this predicate will only allow events for the odigos cluster collectors.
type ClusterCollectorsPredicate struct{}

func (i *ClusterCollectorsPredicate) Create(e event.CreateEvent) bool {
	if e.Object == nil {
		return false
	}
	return hasClusterCollectorLabel(e.Object)
}

func (i *ClusterCollectorsPredicate) Update(e event.UpdateEvent) bool {
	if e.ObjectNew == nil || e.ObjectOld == nil {
		return false
	}
	return hasClusterCollectorLabel(e.ObjectNew)
}

func (i *ClusterCollectorsPredicate) Delete(e event.DeleteEvent) bool {
	if e.Object == nil {
		return false
	}
	return hasClusterCollectorLabel(e.Object)
}

func (i *ClusterCollectorsPredicate) Generic(e event.GenericEvent) bool {
	if e.Object == nil {
		return false
	}
	return hasClusterCollectorLabel(e.Object)
}

var _ predicate.Predicate = &ClusterCollectorsPredicate{}
