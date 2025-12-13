package nodedetails

import (
	odigosv1alpha1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	utilpredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

func SetupWithManager(mgr ctrl.Manager) error {

	err := builder.
		ControllerManagedBy(mgr).
		Named("nodedetails-nodedetails").
		WithEventFilter(utilpredicate.CreationPredicate{}).
		For(&odigosv1alpha1.NodeDetails{}).
		Complete(&NodeDetailsReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
	if err != nil {
		return err
	}

	return nil
}
