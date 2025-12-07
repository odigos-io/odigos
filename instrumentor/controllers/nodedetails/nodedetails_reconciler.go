package nodedetails

import (
	"context"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type NodeDetailsReconciler struct {
	Client client.Client
	Scheme *runtime.Scheme
}

func (r *NodeDetailsReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var nodeDetails odigosv1.NodeDetails
	err := r.Client.Get(ctx, req.NamespacedName, &nodeDetails)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If DiscoveryOdigletPodName is set, delete that pod to trigger a restart
	// The webhook will then prevent the new pod from running discovery again
	if nodeDetails.Spec.DiscoveryOdigletPodName != "" {
		podName := nodeDetails.Spec.DiscoveryOdigletPodName
		podNamespace := req.Namespace

		pod := &corev1.Pod{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: podName, Namespace: podNamespace}, pod)
		if err != nil {
			// If pod not found, it's already been deleted or doesn't exist - nothing to do
			if client.IgnoreNotFound(err) == nil {
				logger.V(1).Info("Odiglet pod not found, already deleted", "pod", podName)
				return ctrl.Result{}, nil
			}
			return ctrl.Result{}, err
		}

		// Pod exists, delete it to trigger a restart without discovery
		logger.Info("Deleting odiglet pod for node details update", "pod", podName, "namespace", podNamespace)
		err = r.Client.Delete(ctx, pod)
		if err != nil {
			logger.Error(err, "Failed to delete odiglet pod", "pod", podName, "namespace", podNamespace)
			return ctrl.Result{}, err
		}

		logger.Info("Successfully deleted odiglet pod", "pod", podName, "namespace", podNamespace)
	}

	return ctrl.Result{}, nil
}
