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

	odigosv1 "github.com/keyval-dev/odigos/api/odigos/v1alpha1"
	"github.com/keyval-dev/odigos/scheduler/controllers/collectorgroups"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// CollectorsGroupReconciler reconciles a CollectorsGroup object
type CollectorsGroupReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=odigos.io,resources=collectorsgroups/finalizers,verbs=update
func (r *CollectorsGroupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	var collectorGroups odigosv1.CollectorsGroupList
	err := r.List(ctx, &collectorGroups, client.InNamespace(req.Namespace))
	if err != nil {
		logger.Error(err, "failed to list collectors groups")
		return ctrl.Result{}, err
	}

	gatewayReady := false
	dataCollectionExists := false
	for _, collectorGroup := range collectorGroups.Items {
		if collectorGroup.Spec.Role == odigosv1.CollectorsGroupRoleGateway && collectorGroup.Status.Ready {
			gatewayReady = true
		}

		if collectorGroup.Spec.Role == odigosv1.CollectorsGroupRoleDataCollection {
			dataCollectionExists = true
		}
	}

	if gatewayReady && !dataCollectionExists {
		logger.Info("creating data collection collector group")
		err = r.Create(ctx, collectorgroups.NewDataCollection(req.Namespace))
		if err != nil {
			logger.Error(err, "failed to create data collection collector group")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CollectorsGroupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&odigosv1.CollectorsGroup{}).
		Complete(r)
}
