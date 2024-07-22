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

	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	appsv1 "k8s.io/api/apps/v1"
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
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching namespace object")
		return ctrl.Result{}, err
	}

	// If namespace is labeled, skip
	if err == nil && workload.IsObjectLabeledForInstrumentation(&ns) {
		return ctrl.Result{}, nil
	}

	// Because of cache settings in the controller, when namespace is unlabelled, it is appearing as not found
	var deps appsv1.DeploymentList
	err = r.Client.List(ctx, &deps, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching deployments")
		return ctrl.Result{}, err
	}

	for _, dep := range deps.Items {
		if !workload.IsObjectLabeledForInstrumentation(&dep) {
			if err := deleteWorkloadInstrumentedApplication(ctx, r.Client, &dep); err != nil {
				logger.Error(err, "error removing runtime details")
				return ctrl.Result{}, err
			}
			err = removeReportedNameAnnotation(ctx, r.Client, &dep)
			if err != nil {
				logger.Error(err, "error removing reported name annotation from deployment")
				return ctrl.Result{}, err
			}
		}
	}

	var ss appsv1.StatefulSetList
	err = r.Client.List(ctx, &ss, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		logger.Error(err, "error fetching statefulsets")
		return ctrl.Result{}, err
	}

	for _, s := range ss.Items {
		if !workload.IsObjectLabeledForInstrumentation(&s) {
			if err := deleteWorkloadInstrumentedApplication(ctx, r.Client, &s); err != nil {
				logger.Error(err, "error removing runtime details")
				return ctrl.Result{}, err
			}
			err = removeReportedNameAnnotation(ctx, r.Client, &s)
			if err != nil {
				logger.Error(err, "error removing reported name annotation from statefulset")
				return ctrl.Result{}, err
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
		if !workload.IsObjectLabeledForInstrumentation(&d) {
			if err := deleteWorkloadInstrumentedApplication(ctx, r.Client, &d); err != nil {
				logger.Error(err, "error removing runtime details")
				return ctrl.Result{}, err
			}
			err = removeReportedNameAnnotation(ctx, r.Client, &d)
			if err != nil {
				logger.Error(err, "error removing reported name annotation from daemonset")
				return ctrl.Result{}, err
			}
		}
	}

	return ctrl.Result{}, nil
}
