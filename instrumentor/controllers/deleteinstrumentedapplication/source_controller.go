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

package deleteinstrumentedapplication

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling Deleted Source object", "name", req.Name, "namespace", req.Namespace)

	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !source.DeletionTimestamp.IsZero() {
		logger.Info("Reconciling workload for deleted Source object", "name", req.Name, "namespace", req.Namespace)
		obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)
		err = r.Client.Get(ctx, types.NamespacedName{Name: source.Spec.Workload.Name, Namespace: source.Spec.Workload.Namespace}, obj)
		if err != nil {
			// TODO: Deleted objects should be filtered in the event filter
			return ctrl.Result{}, err
		}

		if controllerutil.ContainsFinalizer(source, consts.InstrumentedApplicationFinalizer) {
			controllerutil.RemoveFinalizer(source, consts.InstrumentedApplicationFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return ctrl.Result{}, err
			}
		}

		err = reconcileWorkloadObject(ctx, r.Client, obj)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}
