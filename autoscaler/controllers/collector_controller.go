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
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	odigosv1 "github.com/keyval-dev/odigos/api/v1"
)

const (
	CollectorLabel = "odigos.io/collector"
)

var (
	ownerKey     = ".metadata.controller"
	apiGVStr     = odigosv1.GroupVersion.String()
	commonLabels = map[string]string{
		CollectorLabel: "true",
	}
)

// CollectorReconciler reconciles a Collector object
type CollectorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=collectors,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=collectors/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=collectors/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Collector object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *CollectorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Read collector object
	var collector odigosv1.Collector
	err := r.Get(ctx, req.NamespacedName, &collector)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error(err, "error getting collector object")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	// sync child config maps
	cmUpdated, err := r.syncConfigMaps(ctx, &collector)
	if err != nil {
		logger.Error(err, "error syncing config maps")
		return ctrl.Result{}, err
	}
	logger.V(0).Info("synced config maps", "updated", cmUpdated)

	// sync child pods
	podUpdated, err := r.syncPods(ctx, &collector)
	if err != nil {
		logger.Error(err, "error syncing pods")
		return ctrl.Result{}, err
	}
	logger.V(0).Info("synced pods", "updated", podUpdated)

	// sync child services
	svcUpdated, err := r.syncServices(ctx, &collector)
	if err != nil {
		logger.Error(err, "error syncing services")
	}
	logger.V(0).Info("synced services", "updated", svcUpdated)

	// update .status.ready = (cmUpdated && !svcUpdate && !podUpdated (only if different from current status_
	ready := !cmUpdated && !svcUpdated && !podUpdated
	if ready != collector.Status.Ready {
		collector.Status.Ready = ready
		err = r.Status().Update(ctx, &collector)
		if err != nil {
			logger.Error(err, "error updating ready status")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index pods by owner for fast lookup
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Pod{}, ownerKey, func(rawObj client.Object) []string {
		pod := rawObj.(*v1.Pod)
		owner := metav1.GetControllerOf(pod)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Collector" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	// Index child config map
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.ConfigMap{}, ownerKey, func(rawObj client.Object) []string {
		cm := rawObj.(*v1.ConfigMap)
		owner := metav1.GetControllerOf(cm)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Collector" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	// Index child services
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &v1.Service{}, ownerKey, func(rawObj client.Object) []string {
		svc := rawObj.(*v1.Service)
		owner := metav1.GetControllerOf(svc)
		if owner == nil {
			return nil
		}

		if owner.APIVersion != apiGVStr || owner.Kind != "Collector" {
			return nil
		}

		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Collector{}).
		Owns(&v1.Pod{}).
		Owns(&v1.Service{}).
		Owns(&v1.ConfigMap{}).
		Complete(r)
}
