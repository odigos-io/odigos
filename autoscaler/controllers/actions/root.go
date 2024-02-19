package actions

import (
	v1 "github.com/keyval-dev/odigos/api/odigos/action/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := ctrl.NewControllerManagedBy(mgr).
		For(&v1.InsertClusterAttributes{}).
		Complete(&InsertClusterAttributesReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
