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

package deleteinstrumentationconfig

import (
	"context"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
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
	logger.Info("Reconciling Source object", "name", req.Name, "namespace", req.Namespace)

	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if !source.DeletionTimestamp.IsZero() {
		logger.Info("Reconciling workload for deleted Source object", "name", req.Name, "namespace", req.Namespace)
		if source.Spec.Workload.Kind == "Namespace" {
			logger.V(2).Info("Uninstrumenting deployments for Namespace Source", "name", req.Name, "namespace", req.Namespace)
			var deps appsv1.DeploymentList
			err = r.Client.List(ctx, &deps, client.InNamespace(req.Namespace))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching deployments")
				return ctrl.Result{}, err
			}

			for _, dep := range deps.Items {
				logger.V(4).Info("uninstrumenting deployment", "name", dep.Name, "namespace", dep.Namespace)
				err := syncGenericWorkloadListToNs(ctx, r.Client, workload.WorkloadKindDeployment, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
				if err != nil {
					return ctrl.Result{}, err
				}
			}

			logger.V(2).Info("Uninstrumenting statefulsets for Namespace Source", "name", req.Name, "namespace", req.Namespace)
			var ss appsv1.StatefulSetList
			err = r.Client.List(ctx, &ss, client.InNamespace(req.Namespace))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching statefulsets")
				return ctrl.Result{}, err
			}

			for _, s := range ss.Items {
				logger.V(4).Info("uninstrumenting statefulset", "name", s.Name, "namespace", s.Namespace)
				err := syncGenericWorkloadListToNs(ctx, r.Client, workload.WorkloadKindStatefulSet, client.ObjectKey{Namespace: s.Namespace, Name: s.Name})
				if err != nil {
					return ctrl.Result{}, err
				}
			}

			logger.V(2).Info("Uninstrumenting daemonsets for Namespace Source", "name", req.Name, "namespace", req.Namespace)
			var ds appsv1.DaemonSetList
			err = r.Client.List(ctx, &ds, client.InNamespace(req.Namespace))
			if client.IgnoreNotFound(err) != nil {
				logger.Error(err, "error fetching daemonsets")
				return ctrl.Result{}, err
			}

			for _, d := range ds.Items {
				logger.V(4).Info("uninstrumenting daemonset", "name", d.Name, "namespace", d.Namespace)
				err := syncGenericWorkloadListToNs(ctx, r.Client, workload.WorkloadKindDaemonSet, client.ObjectKey{Namespace: d.Namespace, Name: d.Name})
				if err != nil {
					return ctrl.Result{}, err
				}
			}
		} else {
			obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)
			err = r.Client.Get(ctx, types.NamespacedName{Name: source.Spec.Workload.Name, Namespace: source.Spec.Workload.Namespace}, obj)
			if err != nil {
				// TODO: Deleted objects should be filtered in the event filter
				return ctrl.Result{}, err
			}

			err = reconcileWorkloadObject(ctx, r.Client, obj)
			if err != nil {
				return ctrl.Result{}, err
			}
		}

		if controllerutil.ContainsFinalizer(source, consts.InstrumentedApplicationFinalizer) {
			controllerutil.RemoveFinalizer(source, consts.InstrumentedApplicationFinalizer)
			if err := r.Update(ctx, source); err != nil {
				return k8sutils.K8SUpdateErrorHandler(err)
			}
		}
	}
	return ctrl.Result{}, nil
}
