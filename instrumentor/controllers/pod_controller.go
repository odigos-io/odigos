/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"github.com/keyval-dev/odigos/instrumentor/consts"
	v1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=keyval.dev,resources=pods,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=keyval.dev,resources=pods/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=keyval.dev,resources=pods/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Pod object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var pod v1.Pod
	err := r.Get(ctx, req.NamespacedName, &pod)
	if err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			logger.Error(err, "error fetching pod")
		}

		return ctrl.Result{}, err
	}

	if !r.isLangDetectionPod(&pod) {
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == v1.PodSucceeded || pod.Status.Phase == v1.PodFailed {
		err = r.Client.Delete(ctx, &pod)
		if client.IgnoreNotFound(err) != nil {
			logger.Error(err, "failed to delete lang detection pod")
			return ctrl.Result{}, err
		}

		logger.V(0).Info("deleted lang detection pod")
	}

	return ctrl.Result{}, nil
}

func (r *PodReconciler) isLangDetectionPod(pod *v1.Pod) bool {
	val, exists := pod.Annotations[consts.LangDetectionContainerAnnotationKey]
	return exists && val == "true"
}

// SetupWithManager sets up the controller with the Manager.
func (r *PodReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Pod{}).
		Complete(r)
}
