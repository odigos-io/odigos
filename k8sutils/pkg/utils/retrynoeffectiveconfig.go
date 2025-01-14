package utils

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func K8SNoEffectiveConfigErrorHandler(err error) (reconcile.Result, error) {
	if err == ErrOdigosEffectiveConfigNotFound {
		return reconcile.Result{
			Requeue:      true,
			RequeueAfter: 5 * time.Second,
		}, nil
	}
	return reconcile.Result{}, err
}
