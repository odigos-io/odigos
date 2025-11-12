package metricshandler

import (
	"github.com/odigos-io/odigos/api/k8sconsts"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
)

// SetupWithManager registers the CAUpdaterReconciler
// similar to how clustercollector sets up its controllers.
func SetupWithManager(mgr ctrl.Manager) error {

	return builder.
		ControllerManagedBy(mgr).
		Named("metricshandler-ca-sync").
		For(&corev1.Secret{}).
		WithEventFilter(&odigospredicate.ObjectNamePredicate{
			AllowedObjectName: k8sconsts.AutoscalerWebhookSecretName,
		}).
		Complete(&CAUpdaterReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
}
