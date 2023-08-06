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

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update

// Reconcile is responsible for creating InstrumentedApplication objects for every Deployment.
// In addition, Reconcile patch the deployment according to the discovered language and keeps the `instrumented` field
// of InstrumentedApplication up to date with the deployment spec.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dep appsv1.Deployment
	err := r.Get(ctx, req.NamespacedName, &dep)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching deployment object")
		return ctrl.Result{}, err
	}

	if !isObjectLabeled(&dep) {
		// Remove runtime details is exists
		if err := removeRuntimeDetails(ctx, r.Client, req.Namespace, req.Name, dep.Kind, logger); err != nil {
			logger.Error(err, "error removing runtime details")
			return ctrl.Result{}, err
		}
		updated := dep.DeepCopy()
		if removed := removeReportedNameAnnotation(updated); removed {
			patch := client.MergeFrom(&dep)
			if err := r.Patch(ctx, updated, patch); err != nil {
				logger.Error(err, "error removing reported name annotation from deployment")
				return ctrl.Result{}, err
			}
			logger.Info("removed reported name annotation")
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(r)
}

// func removeReportedNameAnnotation(ctx context.Context, kubeClient client.Client, ns string, kind string, name string) error {
// 	var k8sObj client.Object
// 	err := kubeClient.Get(ctx, client.ObjectKey{Namespace: ns, Name: name, Kind: kind}, k8sObj)
// 	if err != nil {
// 		log.FromContext(ctx).Error(err, "error fetching deployment object")
// 		return
// 	}

// 	if _, ok := dep.Annotations[common.RuntimeNameAnnotation]; ok {
// 		delete(dep.Annotations, common.RuntimeNameAnnotation)
// 		if err := kubeClient.Update(ctx, &dep); err != nil {
// 			log.FromContext(ctx).Error(err, "error updating deployment object")
// 			return
// 		}
// 	}
// }
