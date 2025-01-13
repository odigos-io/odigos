package utils

import "sigs.k8s.io/controller-runtime/pkg/client"

// IsTerminating returns true if a client.Object has a non-zero DeletionTimestamp.
// Otherwise, it returns false.
func IsTerminating(obj client.Object) bool {
	return !obj.GetDeletionTimestamp().IsZero()
}
