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
	"errors"

	"github.com/odigos-io/odigos/api/odigos/v1alpha1"
	"github.com/odigos-io/odigos/k8sutils/pkg/consts"
	k8sutils "github.com/odigos-io/odigos/k8sutils/pkg/utils"
	"github.com/odigos-io/odigos/k8sutils/pkg/workload"

	sourceutils "github.com/odigos-io/odigos/k8sutils/pkg/source"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type SourceReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SourcePredicates returns true if the Source object is relevant to starting language detection.
// This means that the Source must be either:
// 1) A normal (non-excluding) Source AND terminating, or
// 2) An excluding Source AND NOT terminating
// In either of these cases, we want to check if workloads should start to be instrumented.
var SourcePredicates = predicate.Funcs{
	UpdateFunc: func(e event.UpdateEvent) bool {
		source := e.ObjectNew.(*v1alpha1.Source)
		return !sourceutils.IsActiveSource(source)
	},

	CreateFunc: func(e event.CreateEvent) bool {
		source := e.Object.(*v1alpha1.Source)
		return !sourceutils.IsActiveSource(source)
	},

	DeleteFunc: func(e event.DeleteEvent) bool {
		return false
	},

	// Allow generic events (e.g., external triggers)
	GenericFunc: func(e event.GenericEvent) bool {
		source := e.Object.(*v1alpha1.Source)
		return !sourceutils.IsActiveSource(source)
	},
}

func (r *SourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling workload for Source object",
		"name", req.Name,
		"namespace", req.Namespace,
		"kind", source.Spec.Workload.Kind,
		"excluded", v1alpha1.IsExcludedSource(source),
		"terminating", k8sutils.IsTerminating(source))

	if source.Spec.Workload.Kind == workload.WorkloadKindNamespace {
		err = errors.Join(err, syncNamespaceWorkloads(ctx, r.Client, req))
	} else {
		// This is a Source for a specific workload, not an entire namespace
		err = errors.Join(err, r.syncWorkload(ctx, source))
	}

	if !v1alpha1.IsExcludedSource(source) &&
		k8sutils.IsTerminating(source) &&
		controllerutil.ContainsFinalizer(source, consts.DeleteInstrumentationConfigFinalizer) {
		controllerutil.RemoveFinalizer(source, consts.DeleteInstrumentationConfigFinalizer)
		if err := r.Update(ctx, source); err != nil {
			return k8sutils.K8SUpdateErrorHandler(err)
		}
	}

	return ctrl.Result{}, client.IgnoreNotFound(err)
}

func (r *SourceReconciler) syncWorkload(ctx context.Context, source *v1alpha1.Source) error {
	// This is a Source for a specific workload, not an entire namespace
	obj := workload.ClientObjectFromWorkloadKind(source.Spec.Workload.Kind)
	err := r.Client.Get(ctx, types.NamespacedName{Name: source.Spec.Workload.Name, Namespace: source.Spec.Workload.Namespace}, obj)
	if err != nil {
		return err
	}

	instrumented, err := sourceutils.IsObjectInstrumentedBySource(ctx, r.Client, obj)
	if err != nil {
		return err
	}
	if !instrumented {
		err = errors.Join(err, deleteWorkloadInstrumentationConfig(ctx, r.Client, obj))
		err = errors.Join(err, removeReportedNameAnnotation(ctx, r.Client, obj))
	}
	return err
}

func syncNamespaceWorkloads(ctx context.Context, k8sClient client.Client, req ctrl.Request) error {
	var err error
	for _, kind := range []workload.WorkloadKind{
		workload.WorkloadKindDaemonSet,
		workload.WorkloadKindDeployment,
		workload.WorkloadKindStatefulSet,
	} {
		err = errors.Join(err, listAndSyncWorkloadList(ctx, k8sClient, req, kind))
	}
	return err
}

func listAndSyncWorkloadList(ctx context.Context,
	k8sClient client.Client,
	req ctrl.Request,
	kind workload.WorkloadKind) error {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Uninstrumenting workloads for Namespace Source", "name", req.Name, "namespace", req.Namespace, "kind", kind)

	workloads := workload.ClientListObjectFromWorkloadKind(kind)
	err := k8sClient.List(ctx, workloads, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	switch obj := workloads.(type) {
	case *appsv1.DeploymentList:
		for _, dep := range obj.Items {
			err = syncGenericWorkloadListToNs(ctx, k8sClient, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	case *appsv1.DaemonSetList:
		for _, dep := range obj.Items {
			err = syncGenericWorkloadListToNs(ctx, k8sClient, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	case *appsv1.StatefulSetList:
		for _, dep := range obj.Items {
			err = syncGenericWorkloadListToNs(ctx, k8sClient, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	}
	return err
}

func syncGenericWorkloadListToNs(ctx context.Context, c client.Client, kind workload.WorkloadKind, key client.ObjectKey) error {
	// it is very important that we make the changes based on a fresh copy of the workload object
	// if a list operation pulled in state and is now slowly iterating over it, we might be working with stale data
	freshWorkloadCopy := workload.ClientObjectFromWorkloadKind(kind)
	workloadGetErr := c.Get(ctx, key, freshWorkloadCopy)
	if workloadGetErr != nil {
		if apierrors.IsNotFound(workloadGetErr) {
			// if the workload been deleted, we don't need to do anything
			return nil
		} else {
			return workloadGetErr
		}
	}

	instrumented, err := sourceutils.IsObjectInstrumentedBySource(ctx, c, freshWorkloadCopy)
	if err != nil {
		return err
	}
	if instrumented {
		return nil
	}

	err = errors.Join(err, deleteWorkloadInstrumentationConfig(ctx, c, freshWorkloadCopy))
	err = errors.Join(err, removeReportedNameAnnotation(ctx, c, freshWorkloadCopy))
	return err
}
