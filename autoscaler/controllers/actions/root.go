package actions

import (
	v1 "github.com/odigos-io/odigos/api/actions/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.AddClusterInfo{}).
		Complete(&AddClusterInfoReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.DeleteAttribute{}).
		Complete(&DeleteAttributeReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	err = ctrl.NewControllerManagedBy(mgr).
		For(&v1.RenameAttribute{}).
		Complete(&RenameAttributeReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
