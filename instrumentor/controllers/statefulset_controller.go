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
	v1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	instAppSSOwnerKey = ".metadata.statefulset.controller"
)

// StatefulSetReconciler reconciles a StatefulSet object
type StatefulSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=statefulsets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the StatefulSet object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *StatefulSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ss appsv1.StatefulSet
	err := r.Get(ctx, req.NamespacedName, &ss)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching statefulset object")
		return ctrl.Result{}, err
	}

	if shouldSkip(ss.Annotations, ss.Namespace) {
		logger.V(5).Info("skipped statefulset")
		return ctrl.Result{}, nil
	}

	err = syncInstrumentedApps(ctx, &req, r.Client, r.Scheme, ss.Status.ReadyReplicas, &ss, &ss.Spec.Template, instAppSSOwnerKey)
	if err != nil {
		logger.Error(err, "error syncing instrumented apps with statefulsets")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *StatefulSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index InstrumentedApps by owner for fast lookup
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.InstrumentedApplication{}, instAppSSOwnerKey, func(rawObj client.Object) []string {
		instApp := rawObj.(*v1.InstrumentedApplication)
		owner := metav1.GetControllerOf(instApp)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != appsv1.SchemeGroupVersion.String() || owner.Kind != "StatefulSet" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.StatefulSet{}).
		Owns(&v1.InstrumentedApplication{}).
		Complete(r)
}
