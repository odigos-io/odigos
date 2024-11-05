package controllers

import (
	"context"

	controllerconfig "github.com/odigos-io/odigos/autoscaler/controllers/controller_config"
	"github.com/odigos-io/odigos/autoscaler/controllers/gateway"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
)

type SecretReconciler struct {
	client.Client
	Scheme           *runtime.Scheme
	ImagePullSecrets []string
	OdigosVersion    string
	Config           *controllerconfig.ControllerConfig
}

type secretPredicate struct {
	predicate.Funcs
}

func (i *secretPredicate) Create(e event.CreateEvent) bool {
	return false
}

func (i *secretPredicate) Update(e event.UpdateEvent) bool {
	oldSecret, oldOk := e.ObjectOld.(*corev1.Secret)
	newSecret, newOk := e.ObjectNew.(*corev1.Secret)

	if !oldOk || !newOk {
		return false
	}

	return oldSecret.ResourceVersion != newSecret.ResourceVersion
}

func (i *secretPredicate) Delete(e event.DeleteEvent) bool {
	return false
}

func (i *secretPredicate) Generic(e event.GenericEvent) bool {
	return false
}

func (r *SecretReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("Reconciling Secret")

	err := gateway.Sync(ctx, r.Client, r.Scheme, r.ImagePullSecrets, r.OdigosVersion, r.Config.MetricsServerEnabled)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SecretReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}).
		WithEventFilter(&secretPredicate{}).
		Complete(r)
}
