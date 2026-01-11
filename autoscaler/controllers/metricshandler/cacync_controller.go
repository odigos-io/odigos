package metricshandler

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	apiregv1 "k8s.io/kube-aggregator/pkg/apis/apiregistration/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/odigos-io/odigos/api/k8sconsts"
)

// CAUpdaterReconciler watches the webhook cert secret and updates the APIService CABundle.
type CAUpdaterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Reconcile ensures the APIService CA bundle stays synced with the Secret's ca.crt.
func (r *CAUpdaterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Only care about the autoscaler cert Secret
	if req.Name != k8sconsts.AutoscalerWebhookSecretName {
		return ctrl.Result{}, nil
	}

	secret := &corev1.Secret{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name:      req.Name,
		Namespace: req.Namespace,
	}, secret); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ca, ok := secret.Data["ca.crt"]
	if !ok || len(ca) == 0 {
		logger.Info("Secret found but missing ca.crt, skipping")
		return ctrl.Result{}, nil
	}

	apiSvc := &apiregv1.APIService{}
	if err := r.Client.Get(ctx, client.ObjectKey{
		Name: NewAPIServiceName,
	}, apiSvc); err != nil {
		logger.Error(err, "Failed to get APIService")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if string(apiSvc.Spec.CABundle) == string(ca) {
		logger.V(1).Info("CA bundle already up-to-date")
		return ctrl.Result{}, nil
	}

	apiSvc.Spec.CABundle = ca
	if err := r.Client.Update(ctx, apiSvc); err != nil {
		logger.Error(err, "Failed to update APIService CABundle")
		return ctrl.Result{}, err
	}

	logger.Info("Updated APIService CABundle successfully")
	return ctrl.Result{}, nil
}
