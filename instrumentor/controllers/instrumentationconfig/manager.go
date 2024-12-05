package instrumentationconfig

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

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

	// Watch for Deployment changes using DeploymentReconciler
	if err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-deployment").
		For(&appsv1.Deployment{}).
		WithEventFilter(workloadReportedNameAnnotationChanged{}).
		Complete(&DeploymentReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}); err != nil {
		return err
	}

	// Watch for StatefulSet changes using StatefulSetReconciler
	if err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-statefulset").
		For(&appsv1.StatefulSet{}).
		WithEventFilter(workloadReportedNameAnnotationChanged{}).
		Complete(&StatefulSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}); err != nil {
		return err
	}

	// Watch for DaemonSet changes using DaemonSetReconciler
	if err := builder.
		ControllerManagedBy(mgr).
		Named("instrumentor-instrumentationconfig-daemonset").
		For(&appsv1.DaemonSet{}).
		WithEventFilter(workloadReportedNameAnnotationChanged{}).
		Complete(&DaemonSetReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		}); err != nil {
		return err
	}

	return nil
}
