package litellm

import (
	"strings"

	"github.com/odigos-io/odigos/api/k8sconsts"
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

func SetupWithManager(mgr ctrl.Manager) error {
	liteLLMPredicate := &odigospredicates.ObjectNamePredicate{AllowedObjectName: k8sconsts.OdigosLiteLLMName}
	credentialPredicate := predicate.NewPredicateFuncs(func(obj client.Object) bool {
		return obj.GetName() == k8sconsts.OdigosLiteLLMName ||
			obj.GetName() == k8sconsts.OdigosLiteLLMRuntimeSecretName ||
			strings.HasPrefix(obj.GetName(), k8sconsts.OdigosLocalUIAICredentialSecretPrefix)
	})

	// the odigos-litellm ConfigMap is watched so it is recreated if deleted, e.g. by a helm
	// upgrade from a chart version that still rendered it.
	return ctrl.NewControllerManagedBy(mgr).
		Named("litellm-config").
		Watches(&corev1.ConfigMap{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(predicate.Or(
			&odigospredicates.OdigosEffectiveConfigMapPredicate,
			liteLLMPredicate))).
		Watches(&corev1.Secret{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(credentialPredicate)).
		Watches(&appsv1.Deployment{}, &handler.EnqueueRequestForObject{}, builder.WithPredicates(liteLLMPredicate, predicate.GenerationChangedPredicate{})).
		Complete(&liteLLMController{
			Client: mgr.GetClient(),
			Scheme: mgr.GetScheme(),
		})
}
