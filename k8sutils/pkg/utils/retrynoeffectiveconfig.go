package utils

import (
	"errors"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func K8SNoEffectiveConfigErrorHandler(err error) (reconcile.Result, error) {
	if errors.Is(err, ErrOdigosEffectiveConfigNotFound) {
		return reconcile.Result{RequeueAfter: 5 * time.Second}, nil
	}
	return reconcile.Result{}, err
}
