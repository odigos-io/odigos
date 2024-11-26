package utils

import (
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// K8SUpdateErrorHandler is a helper function to handle k8s update errors.
// It returns a reconcile.Result and an error
// If the error is a conflict error, it returns a requeue result without an error
// If the error is a not found error, it returns an empty result without an error
// For other errors, it returns the error as is - which will cause a requeue
func K8SUpdateErrorHandler(err error) (reconcile.Result, error) {
	if errors.IsConflict(err) {
		// For conflict errors, requeue without returning an error.
		// this is so that we don't have errors and stack traces in the logs for valid scenario
		return reconcile.Result{Requeue: true}, nil
	}
	if errors.IsNotFound(err) {
		// For not found errors, ignore
		return reconcile.Result{}, nil
	}
	// For other errors, return as is (will log the stack trace)
	return reconcile.Result{}, err
}
