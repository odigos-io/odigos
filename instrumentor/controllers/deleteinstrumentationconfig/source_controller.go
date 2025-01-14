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

	source := &v1alpha1.Source{}
	err := r.Get(ctx, req.NamespacedName, source)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// If this is a regular Source that's being deleted, or a workload Exclusion Source
	// that's being created, try to uninstrument relevant workloads.
	// if (terminating && !exclude) || (!terminating && exclude)
	if k8sutils.IsTerminating(source) != v1alpha1.IsExcludedSource(source) {
		logger.Info("Reconciling workload for Source object",
			"name", req.Name,
			"namespace", req.Namespace,
			"kind", source.Spec.Workload.Kind,
			"excluded", v1alpha1.IsExcludedSource(source),
			"terminating", k8sutils.IsTerminating(source))

		if source.Spec.Workload.Kind == "Namespace" {
			for _, kind := range []workload.WorkloadKind{
				workload.WorkloadKindDaemonSet,
				workload.WorkloadKindDeployment,
				workload.WorkloadKindStatefulSet,
			} {
				err = errors.Join(err, r.listAndSyncWorkloadList(ctx, req, kind))
			}
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

	sources, err := v1alpha1.GetSources(ctx, r.Client, obj)
	if err != nil {
		return err
	}
	if sources.Namespace == nil ||
		(sources.Namespace != nil && k8sutils.IsTerminating(sources.Namespace)) ||
		(sources.Workload != nil && !k8sutils.IsTerminating(sources.Workload) && v1alpha1.IsExcludedSource(source)) {
		// if this workload doesn't have a live Namespace instrumentation, or it has a live exclusion source, uninstrument it
		err = errors.Join(err, deleteWorkloadInstrumentationConfig(ctx, r.Client, obj))
		err = errors.Join(err, removeReportedNameAnnotation(ctx, r.Client, obj))
	}
	return err
}

func (r *SourceReconciler) listAndSyncWorkloadList(ctx context.Context,
	req ctrl.Request,
	kind workload.WorkloadKind) error {
	logger := log.FromContext(ctx)
	logger.V(2).Info("Uninstrumenting workloads for Namespace Source", "name", req.Name, "namespace", req.Namespace, "kind", kind)

	workloads := workload.ClientListObjectFromWorkloadKind(kind)
	err := r.Client.List(ctx, workloads, client.InNamespace(req.Name))
	if client.IgnoreNotFound(err) != nil {
		return err
	}

	switch obj := workloads.(type) {
	case *appsv1.DeploymentList:
		for _, dep := range obj.Items {
			err = r.syncWorkloadList(ctx, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	case *appsv1.DaemonSetList:
		for _, dep := range obj.Items {
			err = r.syncWorkloadList(ctx, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	case *appsv1.StatefulSetList:
		for _, dep := range obj.Items {
			err = r.syncWorkloadList(ctx, kind, client.ObjectKey{Namespace: dep.Namespace, Name: dep.Name})
			if err != nil {
				return err
			}
		}
	}
	return err
}

func (r *SourceReconciler) syncWorkloadList(ctx context.Context,
	kind workload.WorkloadKind,
	key client.ObjectKey) error {
	return syncGenericWorkloadListToNs(ctx, r.Client, kind, key)
}
