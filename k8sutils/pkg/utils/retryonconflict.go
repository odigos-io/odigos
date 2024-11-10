package utils

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func RetryOnConflict(err error) (reconcile.Result, error) {
	if errors.IsConflict(err) {
		// For conflict errors, requeue without returning an error.
		// this is so that we don't have errors and stack traces in the logs for valid scenario
		return reconcile.Result{Requeue: true}, nil
	}
	// For other errors, return as is (will log the stack trace)
	return reconcile.Result{}, err
}
