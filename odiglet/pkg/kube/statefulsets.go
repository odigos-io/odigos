package kube

import (
	"context"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type StatefulSetsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (s *StatefulSetsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ss appsv1.StatefulSet
	err := s.Client.Get(ctx, request.NamespacedName, &ss)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching statefulset object")
		return ctrl.Result{}, err
	}

	return inspectRuntimesOfRunningPods(ctx, &logger, ss.Spec.Selector.MatchLabels, s.Client, s.Scheme, &ss)
}
