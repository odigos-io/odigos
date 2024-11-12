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

	"github.com/odigos-io/odigos/k8sutils/pkg/utils"

	odigosv1 "github.com/odigos-io/odigos/api/odigos/v1alpha1"
	odigospredicates "github.com/odigos-io/odigos/k8sutils/pkg/predicate"
	"github.com/odigos-io/odigos/scheduler/controllers/collectorgroups"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DestinationReconciler reconciles a Destination object
type DestinationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=odigos.io,resources=destinations,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=odigos.io,resources=destinations/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=odigos.io,resources=destinations/finalizers,verbs=update
func (r *DestinationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var dests odigosv1.DestinationList
	err := r.List(ctx, &dests, client.InNamespace(req.Namespace))
	if err != nil {
		logger.Error(err, "failed to list destinations")
		return ctrl.Result{}, err
	}

	if len(dests.Items) > 0 {
		logger.V(0).Info("destinations found, syncing cluster collector group")
		err := utils.ApplyCollectorGroup(ctx, r.Client, collectorgroups.NewClusterCollectorGroup(req.Namespace))
		if err != nil {
			logger.Error(err, "failed to sync cluster collector group")
			return ctrl.Result{}, err
		}
	}
	// once the gateway is created, it is not deleted, even if there are no destinations.
	// we might want to re-consider this behavior.

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DestinationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.Destination{}).
		WithEventFilter(&odigospredicates.ExistencePredicate{}). // only care when destinations are created or deleted
		Complete(r)
}
