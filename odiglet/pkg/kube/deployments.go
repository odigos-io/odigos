package kube

import (
	"context"
	"github.com/keyval-dev/odigos/odiglet/pkg/env"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type DeploymentsReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (d *DeploymentsReconciler) Reconcile(ctx context.Context, request ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(0).Info("reconciling deployment")

	var dep appsv1.Deployment
	err := d.Client.Get(ctx, request.NamespacedName, &dep)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching deployment object")
		return ctrl.Result{}, err
	}

	pods, err := d.getRunningPods(ctx, &dep)
	if err != nil {
		logger.Error(err, "error fetching running pods")
		return ctrl.Result{}, err
	}

	if len(pods) == 0 {
		return ctrl.Result{}, nil
	}

	runtimeResults, err := runtimeInspection(pods)
	if err != nil {
		logger.Error(err, "error inspecting pods")
		return ctrl.Result{}, err
	}

	err = persistRuntimeResults(ctx, runtimeResults, &dep, d.Client, d.Scheme)
	if err != nil {
		logger.Error(err, "error persisting runtime results")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (d *DeploymentsReconciler) getRunningPods(ctx context.Context, dep *appsv1.Deployment) ([]corev1.Pod, error) {
	var podList corev1.PodList
	err := d.Client.List(ctx, &podList, client.MatchingLabels(dep.Spec.Selector.MatchLabels), client.InNamespace(dep.Namespace))

	var filteredPods []corev1.Pod
	for _, pod := range podList.Items {
		if pod.Spec.NodeName == env.Current.NodeName && pod.Status.Phase == corev1.PodRunning {
			filteredPods = append(filteredPods, pod)
		}
	}

	if err != nil {
		return nil, err
	}

	return filteredPods, nil
}
