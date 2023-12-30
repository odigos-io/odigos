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

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *NamespaceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var ns corev1.Namespace
	err := r.Get(ctx, client.ObjectKey{Name: req.Name}, &ns)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		logger.Error(err, "error fetching namespace object")
		return ctrl.Result{}, err
	}

	// If namespace is labeled, skip
	if isInstrumentationLabelEnabled(&ns) {
		return ctrl.Result{}, nil
	}

	var deps appsv1.DeploymentList
	err = r.Client.List(ctx, &deps, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		if !isInstrumentationLabelEnabled(&dep) {
			if err := removeRuntimeDetails(ctx, r.Client, dep.Namespace, dep.Name, dep.Kind, logger); err != nil {
				logger.Error(err, "error removing runtime details")
				return ctrl.Result{}, err
			}
		}
		updated := dep.DeepCopy()
		if removed := removeReportedNameAnnotation(updated); removed {
			patch := client.MergeFrom(&dep)
			if err := r.Patch(ctx, updated, patch); err != nil {
				logger.Error(err, "error removing reported name annotation from deployment")
				return ctrl.Result{}, err
			}
			logger.Info("removed reported name annotation from deployment")
		}
	}

	var ss appsv1.StatefulSetList
	err = r.Client.List(ctx, &ss, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching statefulsets")
		return ctrl.Result{}, err
	}

	for _, s := range ss.Items {
		if !isInstrumentationLabelEnabled(&s) {
			if err := removeRuntimeDetails(ctx, r.Client, s.Namespace, s.Name, s.Kind, logger); err != nil {
				logger.Error(err, "error removing runtime details")
				return ctrl.Result{}, err
			}
			updated := s.DeepCopy()
			if removed := removeReportedNameAnnotation(updated); removed {
				patch := client.MergeFrom(&s)
				if err := r.Patch(ctx, updated, patch); err != nil {
					logger.Error(err, "error removing reported name annotation from statefulset")
					return ctrl.Result{}, err
				}
				logger.Info("removed reported name annotation from stateful set")
			}
		}
	}

	var ds appsv1.DaemonSetList
	err = r.Client.List(ctx, &ds, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching daemonsets")
		return ctrl.Result{}, err
	}

	for _, d := range ds.Items {
		if !isInstrumentationLabelEnabled(&d) {
			if err := removeRuntimeDetails(ctx, r.Client, d.Namespace, d.Name, d.Kind, logger); err != nil {
				logger.Error(err, "error removing runtime details")
				return ctrl.Result{}, err
			}
			updated := d.DeepCopy()
			if removed := removeReportedNameAnnotation(updated); removed {
				patch := client.MergeFrom(&d)
				if err := r.Patch(ctx, updated, patch); err != nil {
					logger.Error(err, "error removing reported name annotation from daemonset")
					return ctrl.Result{}, err
				}
				logger.Info("removed reported name annotation from daemonset set")
			}
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *NamespaceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Namespace{}).
		WithEventFilter(predicate.LabelChangedPredicate{}).
		Complete(r)
}
