package kube

import (
	"context"
	"github.com/keyval-dev/odigos/odiglet/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type PodsReconciler struct {
}

func (p *PodsReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	log.Logger.V(0).Info("Reconciling pods", "request", request)
	return reconcile.Result{}, nil
}
