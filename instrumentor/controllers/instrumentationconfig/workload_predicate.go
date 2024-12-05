package instrumentationconfig

import (
	"github.com/odigos-io/odigos/common/consts"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// workloadReportedNameAnnotationChanged is a custom predicate that detects changes
// to the `odigos.io/reported-name` annotation on workload resources such as
// Deployment, StatefulSet, and DaemonSet. This ensures that the controller
// reacts only when the specific annotation is updated.
type workloadReportedNameAnnotationChanged struct {
	predicate.Funcs
}

// the instrumentation config is create by the instrumented application controller
func (w workloadReportedNameAnnotationChanged) Create(e event.CreateEvent) bool {
	return false
}

func (w workloadReportedNameAnnotationChanged) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	oldAnnotations := e.ObjectOld.GetAnnotations()
	newAnnotations := e.ObjectNew.GetAnnotations()

	oldName := oldAnnotations[consts.OdigosReportedNameAnnotation]
	newName := newAnnotations[consts.OdigosReportedNameAnnotation]

	return oldName != newName
}

func (w workloadReportedNameAnnotationChanged) Delete(e event.DeleteEvent) bool {
	return false
}

func (w workloadReportedNameAnnotationChanged) Generic(e event.GenericEvent) bool {
	return false
}
