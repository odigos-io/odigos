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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/keyval-dev/odigos/cooper/api/v1"
)

// DestinationReconciler reconciles a Destination object
type DestinationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=destinations,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=destinations/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=observability.control.plane.keyval.dev,resources=destinations/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// the Destination object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.2/pkg/reconcile
func (r *DestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dest v1.Destination
	err := r.Get(ctx, req.NamespacedName, &dest)
	if err != nil {
		err = client.IgnoreNotFound(err)
		if err != nil {
			logger.Error(err, "error fetching destination")
		}
		return ctrl.Result{}, err
	}

	collectors, err := r.listCollectors(ctx, &req)
	if err != nil {
		logger.Error(err, "error getting existing collectors")
		return ctrl.Result{}, err
	}

	if len(collectors.Items) == 0 {
		logger.V(0).Info("no running collectors, creating new one")
		err = r.createCollectors(ctx, &req)
		if err != nil {
			logger.Error(err, "error creating new collector")
			return ctrl.Result{}, err
		}
	} else {
		err = r.updateExistingCollectors(ctx, collectors)
		if err != nil {
			logger.Error(err, "failed updating existing collectors")
			return ctrl.Result{}, err
		}
	}

	// TODO: move to pod controller
	//err = r.scheduleAppsToCollectors(collectors)
	//if err != nil {
	//	logger.Error(err, "failed scheduling apps to collectors")
	//	return ctrl.Result{}, err
	//}

	return ctrl.Result{}, nil
}

func (r *DestinationReconciler) listCollectors(ctx context.Context, req *ctrl.Request) (*v1.CollectorList, error) {
	var collectorList v1.CollectorList
	err := r.List(ctx, &collectorList, client.InNamespace(req.Namespace))
	if err != nil {
		return nil, err
	}

	return &collectorList, nil
}

func (r *DestinationReconciler) createCollectors(ctx context.Context, req *ctrl.Request) error {
	return r.Create(ctx, &v1.Collector{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "odigos-collector-",
			Namespace:    req.Namespace,
		},
	})
}

func (r *DestinationReconciler) updateExistingCollectors(ctx context.Context, collectors *v1.CollectorList) error {
	for _, col := range collectors.Items {
		col.Status.Ready = false
		err := r.Status().Update(ctx, &col)
		if err != nil {
			return err
		}
	}

	return nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1.Destination{}).
		Complete(r)
}
