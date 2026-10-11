package metricshandler

import (
	"context"

	"github.com/odigos-io/odigos/api/k8sconsts"
	"github.com/odigos-io/odigos/k8sutils/pkg/env"
	odigospredicate "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// SetupWithManager registers the CAUpdaterReconciler
// similar to how clustercollector sets up its controllers.
func SetupWithManager(mgr ctrl.Manager) error {
	enqueueCertSecret := handler.EnqueueRequestsFromMapFunc(func(context.Context, client.Object) []reconcile.Request {
		return []reconcile.Request{{
			NamespacedName: types.NamespacedName{
				Name:      k8sconsts.InstrumentorWebhookSecretName,
				Namespace: env.GetCurrentNamespace(),
			},
		}}
	})

	return builder.
		ControllerManagedBy(mgr).
		Named("metricshandler-ca-sync").
		For(&corev1.Secret{}, builder.WithPredicates(&odigospredicate.ObjectNamePredicate{
			AllowedObjectName: k8sconsts.InstrumentorWebhookSecretName,
		})).
		Watches(&apiregv1.APIService{}, enqueueCertSecret, builder.WithPredicates(&odigospredicate.ObjectNamePredicate{
			AllowedObjectName: k8sconsts.CustomMetricsAPIServiceName,
		})).
		Complete(&CAUpdaterReconciler{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
}
