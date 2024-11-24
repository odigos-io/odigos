package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/common/consts"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
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

func (w workloadReportedNameAnnotationChanged) Create(e event.CreateEvent) bool {
	return true
}

func (w workloadReportedNameAnnotationChanged) Update(e event.UpdateEvent) bool {
	if e.ObjectOld == nil || e.ObjectNew == nil {
		return false
	}

	// Compare annotations
	oldAnnotations := e.ObjectOld.GetAnnotations()
	newAnnotations := e.ObjectNew.GetAnnotations()

	// Check if the `odigos.io/reported-name` annotation has changed
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

func SetupWithManager(mgr ctrl.Manager) error {
	// Watch InstrumentationRule
	err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentationrule").
		For(&odigosv1alpha1.InstrumentationRule{}).
		Complete(&InstrumentationRuleReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Watch InstrumentedApplication
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-instrumentedapplication").
		For(&odigosv1alpha1.InstrumentedApplication{}).
		Complete(&InstrumentedApplicationReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Watch for Deployment changes
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-workloads-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(workloadReportedNameAnnotationChanged{}).
		Complete(&WorkloadsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Watch for StatefulSet changes
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-workloads-statefulset").
		For(&appsv1.StatefulSet{}).
		WithEventFilter(workloadReportedNameAnnotationChanged{}).
		Complete(&WorkloadsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	// Watch for DaemonSet changes
	err = builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-workloads-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(workloadReportedNameAnnotationChanged{}).
		Complete(&WorkloadsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
